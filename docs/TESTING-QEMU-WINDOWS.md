# Testing kacuOS in QEMU on Windows

You do **not** need a Linux machine to *run* kacuOS вЂ” only to *build* the ISO
(`archiso` is Arch-specific). This guide covers both: building the ISO in CI
and running it with QEMU on Windows.

---

## 1. Get an ISO

### Option A вЂ” GitHub Actions (recommended, no Linux needed)

The repository contains `.github/workflows/build-iso.yml`.

1. Push the repository to GitHub.
2. Open **Actions в†’ Build kacuOS ISO в†’ Run workflow**.
3. When the job finishes, open the run and download the **`kacuos-iso`**
   artifact.
4. Unzip it вЂ” you get something like `kacuos-2026.10.21-x86_64.iso`.

### Option B вЂ” Docker Desktop with an Arch container

Requires WSL2 and loop-device support. From the repository root:

```powershell
docker run --rm -it --privileged -v ${PWD}:/work -w /work archlinux:latest bash
```

Inside the container:

```bash
pacman -Syu --noconfirm archiso git python rust gcc make
cd archiso
python bake.py
python setup_boot_files.py kacu-desktop
cd kacu-desktop
mkarchiso -v -o out/ .
exit
```

The ISO appears in `archiso/kacu-desktop/out/`.

> If `losetup`/`mkarchiso` fails inside Docker, use Option A or a real Arch VM.

---

## 2. Install QEMU on Windows

1. Download a Windows build:
   * <https://qemu.weilnetz.de/w64/> (installer), or
   * <https://www.qemu.org/download/#windows>
2. Install to the default `C:\Program Files\qemu`.
3. Add `C:\Program Files\qemu` to **PATH** (optional if you use
   `tools/run_qemu.py`, which searches the default directory itself).
4. Verify in PowerShell:

```powershell
qemu-system-x86_64 --version
qemu-img --version
```

### Enable hardware acceleration (WHPX)

Without acceleration a GNOME desktop in QEMU is very slow. Enable Windows
Hypervisor Platform once:

```powershell
dism /online /enable-feature /featurename:HypervisorPlatform /all /norestart
dism /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart
```

Reboot, then check:

```powershell
qemu-system-x86_64 -accel help
```

You should see `whpx`. If not, use `--accel tcg` (slower but always works).

---

## 3. Test the live ISO (BIOS)

The helper script builds the right command line for you:

```powershell
cd E:\kacuOS\kacuOS
python tools\run_qemu.py --iso "C:\isos\kacuos-2026.10.21-x86_64.iso" --memory 4096 --cpus 2
```

Equivalent manual command:

```powershell
& "C:\Program Files\qemu\qemu-system-x86_64.exe" `
  -machine q35 -m 4096 -smp 2 -accel whpx `
  -cdrom "C:\isos\kacuos-2026.10.21-x86_64.iso" -boot d `
  -device virtio-vga -nic user,model=virtio-net-pci `
  -usb -device usb-tablet -rtc base=localtime -display gtk
```

What you should see:

1. systemd boot messages, then GDM.
2. Automatic login as `liveuser` into a GNOME Wayland session.
3. A working desktop, network via DHCP, and the **Install kacuOS** icon.

Quick checks inside the live session (GNOME Console):

```bash
zramctl                     # zram0 should be active, ~4G, zstd
swapon --show               # /dev/zram0
systemctl status gdm
systemctl is-enabled tracker-miner-fs-3.service   # masked
kacu-pkg list
kacu-sysinfo
ping -c 3 archlinux.org
```

If the desktop does not appear, switch to a TTY with `Ctrl+Alt+F2`
(login `root`, no password) and inspect:

```bash
journalctl -b -u gdm
journalctl -b -u systemd-zram-setup@zram0
```

---

## 4. Test the live ISO (UEFI)

Use the same command with `--uefi`:

```powershell
python tools\run_qemu.py --iso "C:\isos\kacuos.iso" --uefi --memory 4096 --cpus 2
```

Manual UEFI command (OVMF files ship with QEMU):

```powershell
Copy-Item "C:\Program Files\qemu\share\edk2-i386-vars.fd" .\kacuos-vars.fd -Force

& "C:\Program Files\qemu\qemu-system-x86_64.exe" `
  -machine q35 -m 4096 -smp 2 -accel whpx `
  -drive if=pflash,format=raw,readonly=on,file="C:\Program Files\qemu\share\edk2-x86_64-code.fd" `
  -drive if=pflash,format=raw,file=.\kacuos-vars.fd `
  -cdrom "C:\isos\kacuos.iso" -boot d `
  -device virtio-vga -nic user,model=virtio-net-pci `
  -usb -device usb-tablet -rtc base=localtime -display gtk
```

---

## 5. Test the installer (`kacu_install`)

Create a virtual disk and attach it **together with** the ISO:

```powershell
python tools\run_qemu.py `
  --iso "C:\isos\kacuos.iso" `
  --disk "C:\vm\kacuos.qcow2" --create-disk 20G `
  --memory 4096 --cpus 2
```

Inside the live session:

```bash
sudo kacu_install --list-disks
```

With a virtio disk, QEMU exposes it as `/dev/vda` (the ISO is `/dev/sr0`).
Then either run the installer interactively or drive it directly:

```bash
sudo kacu_install --disk /dev/vda --hostname kacuos --username kacu \
  --timezone Europe/Kyiv --boxes dev --dry-run
```

`--dry-run` prints the plan and stops. When it looks right, drop `--dry-run`
and type `YES` at the confirmation. The installer will:

* create `/dev/vda1` (ESP, UEFI) or a 1 MiB BIOS boot partition,
* create `/dev/vda2` (ext4 root),
* `pacstrap` the GNOME system,
* install GRUB (UEFI or BIOS, matching the firmware you booted QEMU with),
* copy `kacu-setup`/`kacu-pkg` and the zram/dconf configuration,
* install the `dev` box (and any others you selected).

A full install on WHPX typically takes 5вЂ“15 minutes depending on the network.

> **Important:** boot QEMU in the same firmware mode you want the installed
> system to use. If you booted the ISO in BIOS mode, QEMU's NVRAM will not see
> a UEFI GRUB entry, and vice versa.

When it finishes:

```bash
sudo poweroff
```

---

## 6. Boot the installed system

Start QEMU with only the disk (no ISO):

```powershell
python tools\run_qemu.py --disk "C:\vm\kacuos.qcow2" --memory 4096 --cpus 2
```

For a UEFI install add `--uefi` (reusing the same `kacuos-vars.fd` is fine).
You should get GDM, log in as the user you created, and land in GNOME.

Useful post-install checks:

```bash
zramctl
systemctl status gdm NetworkManager
kacu-pkg status
kacu-pkg install ai
```

---

## 7. QEMU controls and troubleshooting

| Problem | Fix |
|---|---|
| `Could not access KVM kernel module` | KVM does not exist on Windows; use `--accel whpx` or `--accel tcg`. |
| `whpx: injection failed` / WHPX missing | Enable Windows Hypervisor Platform and reboot, or use `--accel tcg`. |
| `-display gtk` fails | Use `--display sdl` or `--display default`. |
| Mouse/keyboard captured | Press `Ctrl+Alt+G` to release. |
| Need the QEMU monitor | `Ctrl+Alt+2` (monitor), `Ctrl+Alt+1` (back to display). |
| Quit QEMU | Close the window, or type `quit` in the monitor. |
| Boot drops to UEFI shell | Your ISO/disk was installed in BIOS mode; reinstall with `--uefi`, or boot without UEFI. |
| Very slow desktop | Enable WHPX; make sure `--memory 4096`; close other VMs. |
| No network | Remove `--no-net`; test with `ping archlinux.org`; QEMU user networking is NAT. |
| Disk full | Create a larger `--create-disk` image (e.g. `30G`). |
| ISO won't boot | Check `-boot d`, and that the ISO path has no unusual characters. |
| Need a screenshot | Monitor (`Ctrl+Alt+2`): `screendump C:\vm\shot.ppm`. |

### Snapshot mode

`--snapshot` makes all disk writes temporary вЂ” perfect for testing the
installer repeatedly without recreating the image:

```powershell
python tools\run_qemu.py --iso "C:\isos\kacuos.iso" --disk "C:\vm\kacuos.qcow2" --snapshot
```

---

## 8. What "success" looks like

* ISO boots to GDM and autologins to a GNOME Wayland session.
* `zramctl` shows a `zram0` device using ~4 GiB with `zstd`.
* `tracker-miner-fs-3` is masked and does not run.
* `kacu-pkg list` shows `dev`, `server`, `ai`.
* `sudo kacu_install --dry-run ...` prints a coherent plan.
* A real install reboots into the same GNOME desktop from the virtual disk.
