# kacu-install

Interactive installers for kacuOS. Both run **inside the live ISO** as root and
turn a whole disk into a bootable kacuOS system (Arch Linux + minimal GNOME).

* **`langley/`** — the Go installer (archinstall-style wizard with network
  setup, GPU drivers, audio and software selection).
* **`kacu_install`** — the earlier Python installer (disk → pacstrap → GRUB).

## langley (Go)

The master flow is wizard-style, archinstall-like:

```
Step 1/8 — Network          connectivity check, Ethernet (DHCP) or Wi-Fi (nmcli scan + passphrase)
Step 2/8 — Disk and firmware  disk picker (whole disks only), UEFI/BIOS autodetect, GPT layout
Step 3/8 — System identity    hostname, username, timezone
Step 4/8 — Graphics drivers   lspci/sysfs detection; Intel, AMD, NVIDIA (or DKMS), Mesa only
Step 5/8 — Audio              PipeWire (default), PulseAudio or ALSA only; lists detected cards
Step 6/8 — Software           browser (Firefox / Chromium / Brave via AUR), Steam, extra apps, kacu-pkg boxes
Step 7/8 — Plan               every choice printed, `YES` confirmation, --dry-run supported
Step 8/8 — Installing         sgdisk → mkfs → pacstrap → fstab → config → GRUB → AUR → boxes
```

Extra behaviour worth knowing:

* Steam automatically uncomments `[multilib]` in the target `pacman.conf`
  *before* `pacstrap` (via `-C`) and redeploys it into the target afterwards,
  plus pulls the `lib32-*` Vulkan/Mesa companions for the chosen GPU.
* Brave comes from the AUR: langley puts `paru` into the target (release
  binary, with a fallback warning), gives the new user temporary passwordless
  sudo only for the build and removes it afterwards.
* The CPU's microcode (`intel-ucode`/`amd-ucode`) is validated against
  `/proc/cpuinfo`; the audio choice removes the other stack from the base list
  (selecting PulseAudio drops pipewire/wireplumber and vice versa).
* Passwords are fed over stdin (`chpasswd`) and Wi-Fi passphrases are masked
  in the command log; input is hidden with termios on Linux.
* Logs: `/var/log/langley.log`. The selected disk is always wiped after an
  explicit plan + `YES` (or `--yes`).

### Build

```bash
cd kacu-install/langley
go build -o langley .                  # for the host
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-s -w" -o langley .   # static binary for the ISO
```

No third-party Go dependencies — plain stdlib, so it cross-compiles anywhere.

### Bake into the ISO

Build the static Linux binary in CI and drop it into the airootfs:

```
archiso/kacu-desktop/airootfs/usr/local/bin/langley        (0755)
```

Then either point `kacu-install.desktop` at it:

```
Exec=kgx -e "sudo /usr/local/bin/langley"
```

or keep the launcher as-is and run `sudo langley` from a terminal.

### Usage

```bash
sudo langley                          # fully interactive
sudo langley --list-disks
sudo langley --disk /dev/sda
sudo langley --disk /dev/sda --yes --dry-run
sudo langley --disk /dev/nvme0n1 --uefi --hostname kacuos --username ivan \
             --timezone Europe/Kyiv --boxes dev,ai
```

| Option | Meaning |
|---|---|
| `--disk PATH` | target whole disk (`/dev/sda`, `/dev/nvme0n1`, loop images for tests) |
| `--hostname NAME` | default `kacuos` |
| `--username NAME` | default `kacu` |
| `--timezone TZ` | default `UTC` (e.g. `Europe/Kyiv`) |
| `--boxes LIST` | comma-separated `dev,server,ai` |
| `--uefi` / `--bios` | force the firmware mode |
| `--yes` | skip the `YES` confirmation |
| `--dry-run` | print the plan, change nothing |
| `--list-disks` | list candidate disks and exit |
| `--version` | print the version and exit |

## kacu_install (Python)

The original installer, kept as fallback. Same destructive flow but without
the network/GPU/audio/software wizards — packages come from
`install-packages.txt` and boxes from `--boxes`.

```bash
sudo kacu_install                 # fully interactive
sudo kacu_install --disk /dev/sda --yes --dry-run
sudo kacu_install --disk /dev/nvme0n1 --uefi --boxes dev,ai
```

## Safety (both installers)

* Whole disks only; disks with mounted partitions are refused.
* The plan is always printed and `YES` is required unless `--yes` is given.
* `--dry-run` shows the plan and stops before partitioning.
* `/mnt` is unmounted in a deferred cleanup, also on failures.

Logs: `/var/log/langley.log` and `/var/log/kacu_install.log`.
