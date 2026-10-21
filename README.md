# kacuOS

An Arch-based Linux distribution with a **minimal, low-RAM GNOME (Wayland)**
desktop. Built with `archiso`, tuned for old hardware (Intel Core 2 Duo class,
HDD, 4 GB RAM) using zram and disabled background services.

The customization layer is Python-first:

| Tool | Purpose |
|---|---|
| `kacu_install` | interactive disk installer (Arch + minimal GNOME + GRUB) |
| `kacu-pkg` | package "box" manager (`dev`, `server`, `ai`) |
| `kacu-setup` | applies zram, GNOME/dconf defaults and masks heavy services |
| `kacuctl` | legacy Rust CLI: presets + sandboxed Lua plugins |
| `kacu-sysinfo` | tiny C utility: RAM/CPU/load from `/proc` |

**Plugin guide:** [PLUGINS.md](./PLUGINS.md)
**Test on Windows:** [docs/TESTING-QEMU-WINDOWS.md](./docs/TESTING-QEMU-WINDOWS.md)

## Repository structure

```
archiso/
├── bake.py                  # build tools + copy them into every profile
├── setup_boot_files.py      # copy syslinux/efiboot/grub from releng
├── kacu-desktop/            # main profile: minimal GNOME (Wayland)
├── kacu-server/             # headless profile
└── kacu-desktop-sway/       # legacy sway profile (kept for reference)
kacu-install/                # the Python installer (kacu_install)
kacu-pkg/                    # Python box manager + TOML presets
kacuctl/                     # Rust CLI (legacy presets/plugins)
kacu-sysinfo/                # C system-info helper
plugins-example/             # example Lua plugin
tools/run_qemu.py            # cross-platform QEMU launcher (Windows-friendly)
docs/                        # landing page + QEMU testing guide
```

> `profiledef.sh` and `profile.d/*.sh` are intentionally still shell: archiso
> **sources** `profiledef.sh`, and login shells **source** `profile.d`. They are
> declarative glue, not logic; all logic lives in Python.

## Quick start — tooling only (no ISO)

```bash
cd kacu-pkg && ./kacu-pkg list
```

On Arch, bake the tools into an ISO profile:

```bash
cd archiso
python bake.py --skip-build          # or omit --skip-build to compile Rust/C
python setup_boot_files.py kacu-desktop
cd kacu-desktop
sudo mkarchiso -v -o out/ .
```

## Using the installed system

```bash
sudo kacu-setup --system   # zram + dconf defaults + service masking
kacu-setup --user          # apply GNOME settings to the current user
kacu-pkg list
kacu-pkg install dev
sudo kacu_install          # only from the live ISO
```

## Building the ISO

Requires Arch Linux (native, VM, WSL2-Arch or CI — `archiso` only runs on Arch).

```bash
sudo pacman -S archiso
cd archiso
python bake.py
python setup_boot_files.py kacu-desktop   # keeps the existing profiledef.sh
cd kacu-desktop
sudo mkarchiso -v -o out/ .
```

If you only have Windows, use the GitHub Actions workflow
`.github/workflows/build-iso.yml` (builds and uploads the ISO as an artifact),
then follow [docs/TESTING-QEMU-WINDOWS.md](./docs/TESTING-QEMU-WINDOWS.md).

## License

MIT — see [LICENSE](./LICENSE).
