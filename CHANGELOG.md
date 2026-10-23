# Changelog

## Unreleased — kacuOS refactor

- Renamed the whole project: `nanOS` → `kacuOS`, `nanos-*` → `kacu-*`,
  `nanctl` → `kacuctl`, `nanos-sysinfo` → `kacu-sysinfo`; repository root is
  now `kacuOS/`.
- Converted all logic scripts from Bash to Python:
  `bake.py`, `setup_boot_files.py`, `kacu-setup`, `customize_airootfs.sh`
  (Python content), preset hooks (`.py`), `welcome.py`.
  Only `profiledef.sh` (sourced by mkarchiso) and `profile.d/*.sh`
  (sourced by login shells) remain shell by necessity.
- New archiso profile `archiso/kacu-desktop` (minimal GNOME/Wayland, zram,
  masked tracker/evolution, dconf defaults). The old sway profile is kept as
  `archiso/kacu-desktop-sway`.
- New `kacu-pkg` (Python) box manager with `dev`, `server`, `ai` presets and
  pre/post-install hooks (`.py`, `.lua`, legacy `.sh`).
- New `kacu_install` (Python) disk installer: UEFI/BIOS, GPT, `pacstrap`,
  GRUB, user creation, optional boxes, `--dry-run` and logging.
- Added `.github/workflows/build-iso.yml` to build the ISO in CI and
  `docs/TESTING-QEMU-WINDOWS.md` + `tools/run_qemu.py` for Windows testing.

## v1.0 — kacuctl

First stable release. Scope: `kacuctl` as a standalone preset/plugin
manager for Arch Linux.

**What's in:**
- `kacuctl` (Rust CLI) — `install`, `list`, `status`, `plugin add/list/run-hook`
- Lua plugin engine (`mlua`), sandboxed: `kacu.print`, `kacu.exec_readonly`
  (whitelisted commands only), `kacu.exec` (permission-gated, shown to the
  user before install). Verified against a plugin attempting an
  unauthorized command — correctly blocked.
- `kacu-sysinfo` (C) — reads RAM/CPU/load directly from `/proc`, zero
  deps, wired into `kacuctl status`
- Presets: `devtools` (gcc/cargo/make/cmake), `browser` (Firefox),
  `editor` (VS Code), `cluster` (Docker + k3s, traefik/servicelb/
  metrics-server disabled by default)
- `kacuctl install` runs `pacman -Sy` before installing, so it works on a
  freshly installed system with no synced package database

**Not in this release:**
- The kacuOS live ISO (`archiso/`) — boots inconsistently, hit a kernel
  panic during testing. Kept in the repo as experimental/unfinished;
  `kacuctl` itself has no dependency on it and installs fine on any
  existing Arch system.
- Terminal configurator website

## Unreleased

- kacuOS branding: /etc/os-release rewritten to identify as kacuOS (fixes
  login banner showing "Arch Linux"), applied via customize_airootfs.sh
  to reliably overwrite the os-release symlink
- neofetch added, configured with a custom kacuOS ASCII logo
  (/etc/xdg/neofetch/config.conf, /etc/kacu/kacu-ascii.txt), runs
  automatically on every interactive login via /etc/profile.d
- PLUGINS.md: full guide for writing kacuctl plugins
- docs/index.html: landing page, terminal-styled, ready for GitHub Pages
