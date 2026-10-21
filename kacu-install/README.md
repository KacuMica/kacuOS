# kacu_install

Interactive installer for kacuOS. It runs **inside the live ISO** as root and
turns a whole disk into a bootable kacuOS system (Arch Linux + minimal GNOME).

## Usage

```bash
sudo kacu_install                 # fully interactive
sudo kacu_install --list-disks
sudo kacu_install --disk /dev/sda
sudo kacu_install --disk /dev/sda --yes --dry-run
sudo kacu_install --disk /dev/nvme0n1 --uefi --boxes dev,ai
```

From the live GNOME session you can also click **Install kacuOS** in the
application grid (launches the same command in GNOME Console).

## What it does

1. Lists whole disks (`lsblk`) and asks which one to erase.
2. Detects UEFI vs BIOS (`/sys/firmware/efi`).
3. Creates a GPT layout with `sgdisk`:
   * UEFI — 512 MiB ESP (`ef00`) + root (`8300`, ext4);
   * BIOS — 1 MiB `bios_grub` (`ef02`) + root (ext4).
4. Formats and mounts everything under `/mnt`.
5. `pacstrap` installs `install-packages.txt` (GNOME, audio, fonts, dev basics).
6. Writes `fstab`, locale, timezone, hostname, users and sudoers.
7. Copies `kacu-setup`, `kacu-pkg`, presets, zram/dconf configs into the target
   and runs `kacu-setup --system` inside the chroot.
8. Installs GRUB for UEFI or BIOS and regenerates the initramfs.
9. Optionally runs `kacu-pkg install <boxes>`.

## Safety

* The installer refuses partitions, missing devices and disks with mounted
  partitions.
* It always prints the plan and requires you to type `YES` unless `--yes` is
  given.
* `--dry-run` shows the plan and stops before partitioning.

## Options

| Option | Meaning |
|---|---|
| `--disk PATH` | target whole disk |
| `--hostname NAME` | default `kacuos` |
| `--username NAME` | default `kacu` |
| `--timezone TZ` | default `UTC` (e.g. `Europe/Kyiv`) |
| `--boxes LIST` | comma-separated `dev,server,ai` |
| `--uefi` / `--bios` | force the firmware mode |
| `--yes` | skip the `YES` confirmation |
| `--dry-run` | print the plan, change nothing |
| `--list-disks` | list candidate disks and exit |

Logs are written to `/var/log/kacu_install.log`.
