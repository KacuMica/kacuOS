# kacuOS

A terminal-first, Arch-based Linux distribution: no desktop environment by
default, small RAM footprint, `nanctl` as the customization layer (TOML
presets + sandboxed Lua plugins).

**Website:** https://neofetch-tech.github.io/nanos/ (once GitHub Pages is
enabled — see below)
**Plugin guide:** [PLUGINS.md](./PLUGINS.md)

## Repository structure

```
archiso/         Arch ISO profiles (Server = headless, Desktop = + sway)
nanctl/          the CLI (Rust) — presets, plugins, status
nanos-sysinfo/   small C utility: RAM/CPU/load from /proc, used by `nanctl status`
plugins-example/ example Lua plugin (git branch in your shell prompt)
docs/            landing page (docs/index.html), served via GitHub Pages
PLUGINS.md       guide for writing nanctl plugins
CHANGELOG.md
```

## Quick start — nanctl only (no ISO needed)

```bash
git clone https://github.com/neofetch-tech/nanos.git
cd nanos/nanctl
cargo build --release
sudo cp target/release/nanctl /usr/local/bin/

cd ../nanos-sysinfo
make
sudo make install

nanctl list
nanctl status
```

## Building the ISO

Requires Arch Linux (native, VM, or WSL — `archiso` is unavailable on other
distros).

```bash
sudo pacman -S archiso
cd archiso
./bake-binaries.sh                    # builds + copies nanctl, nanos-sysinfo,
                                       # presets into airootfs
./setup-boot-files.sh nanos-server    # bootloader configs, profiledef.sh,
                                       # passwordless root, mkinitcpio hooks —
                                       # safe to re-run
cd nanos-server
sudo mkarchiso -v -o out/ .
```

Test in QEMU:
```bash
qemu-system-x86_64 -m 1G -cdrom out/nanos-server-*.iso -boot d -display gtk
```

## Enabling the website (GitHub Pages)

Settings → Pages → Source: `Deploy from a branch` → Branch: `main`,
folder: `/docs` → Save. The site will be live at
`https://neofetch-tech.github.io/nanos/` within a couple of minutes.

## License

MIT — see [LICENSE](./LICENSE).
