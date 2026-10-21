# kacu-desktop — archiso profile (minimal GNOME)

Live/install ISO for kacuOS: Arch Linux + minimal GNOME (Wayland), tuned for
a Core 2 Duo / HDD / 4 GB RAM machine.

## What is in here

```
kacu-desktop/
├── packages.x86_64                 # minimal GNOME live environment
├── profiledef.sh                   # ISO metadata + executable-bit map (shell:
│                                   #   archiso sources this file)
├── pacman.conf                     # repos used by mkarchiso
└── airootfs/                       # copied verbatim into the ISO
    ├── root/customize_airootfs.sh  # Python content (archiso runs the file)
    ├── etc/
    │   ├── kacu/gnome_settings.ini # dconf keyfile (GNOME defaults)
    │   ├── kacu/install-packages.txt
    │   ├── pacman.d/mirrorlist
    │   ├── sysctl.d/99-zram.conf
    │   └── systemd/zram-generator.conf
    ├── usr/local/bin/kacu-setup    # Python: zram + dconf + service masking
    └── usr/share/applications/kacu-install.desktop
```

`kacu-pkg`, `kacu_install` and the presets are baked in by `../bake.py`:

```
airootfs/usr/local/bin/kacu-pkg
airootfs/usr/local/bin/kacu_install
airootfs/etc/kacu/presets/{dev,server,ai}.toml
```

## Build

On an Arch Linux host with `archiso` installed:

```bash
sudo pacman -S archiso

cd archiso
python bake.py                          # builds kacuctl/kacu-sysinfo + copies tools
python setup_boot_files.py kacu-desktop # syslinux/efiboot/grub from releng

cd kacu-desktop
sudo mkarchiso -v -o out/ .
```

> `setup_boot_files.py` only writes `profiledef.sh` when it is missing (or with
> `--force-profiledef`), so the hand-maintained file permissions for
> `kacu-setup`, `kacu-pkg`, `kacu_install` and the `.py` hooks are preserved.

## Live session

The build creates a `liveuser` (passwordless sudo, GDM autologin). After boot
you land directly in a GNOME Wayland session. Launch **Install kacuOS** from
the app grid, or run `sudo kacu_install` in a terminal.

## On an installed system

```bash
sudo kacu-setup --system    # zram + dconf defaults + mask heavy services
kacu-setup --user           # apply GNOME settings to the current user
kacu-pkg list
kacu-pkg install dev
```
