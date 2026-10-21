# kacuctl

A preset + plugin manager for Arch Linux, written in Rust.

`kacuctl` installs curated bundles of `pacman` packages ("presets") from
simple TOML files, and lets you extend your setup with sandboxed Lua
plugins — without writing raw `pacman` commands or trusting arbitrary
shell scripts from the internet.

## Why

Setting up a new Arch install usually means either running a long list of
`pacman -S` commands from memory, or copy-pasting someone else's install
script and hoping it doesn't do anything you didn't expect. `kacuctl` gives
you a middle ground: declarative presets you can read before you run them,
and a plugin system where permissions are shown to you *before* install,
not buried in the source.

## Install

```bash
git clone https://github.com/kacu-os/kacuOS.git
cd kacu/kacuctl
cargo build --release
sudo cp target/release/kacuctl /usr/local/bin/
```

Requires `gcc` (kacuctl vendors and compiles its own Lua interpreter on
build, no system Lua dependency needed).

### Or run it in Docker (no Rust toolchain needed locally)

```bash
docker build -t kacuctl .
docker run --rm kacuctl list
docker run --rm kacuctl status
```

Note: `kacuctl install <preset>` calls `pacman`, which only exists on Arch —
the Docker image is Debian-based, so that specific command won't work
inside the container. Everything else (`list`, `status`, `plugin`) works
fine, and it's a quick way to confirm the build is healthy without
installing anything locally.

## Usage

### Presets

A preset is a TOML file describing a group of packages and an optional
post-install hook:

```toml
# presets/devtools.toml
[preset]
name = "devtools"
description = "GCC, cargo, build essentials"
profile = ["desktop", "server"]

[packages]
pacman = ["gcc", "cargo", "make", "cmake", "pkg-config"]
```

```bash
kacuctl list                       # see available presets
kacuctl install devtools           # runs pacman -Sy + the package list
kacuctl install devtools --dry-run # show what would run, don't execute
```

By default `kacuctl` looks for presets in `/etc/kacu/presets`, falling
back to `./presets` for local development. Override with
`KACUOS_PRESETS_DIR`.

Included presets: `devtools`, `browser` (Firefox), `editor` (VS Code),
`cluster` (Docker + k3s, with `traefik`/`servicelb`/`metrics-server`
disabled by default for a smaller footprint).

### Plugins (Lua, sandboxed)

Plugins are Lua scripts that hook into `kacuctl` events. Each plugin ships
a manifest declaring exactly what it's allowed to do:

```toml
# plugin.toml
[plugin]
name = "git-status-prompt"
version = "0.1.0"

[permissions]
exec = false       # can this plugin run arbitrary shell commands?
network = false
hooks = ["shell_prompt"]
```

```lua
-- init.lua
kacu_hooks["shell_prompt"] = function()
    local branch = kacu.exec_readonly("git branch --show-current")
    kacu.print("current branch: " .. branch)
end
```

`kacu.exec_readonly` only allows a small whitelist of read-only commands
(git, ls, cat, pwd, whoami, date, uname) regardless of what the plugin
requests. `kacu.exec` (arbitrary commands) is only wired up if the
plugin's manifest declares `permissions.exec = true` — and `kacuctl`
shows you that request before installing, so you can say no.

```bash
kacuctl plugin add ./path/to/plugin   # shows requested permissions, asks to confirm
kacuctl plugin list
kacuctl plugin run-hook shell_prompt  # manually fire a hook (mostly for testing)
```

### System info

```bash
kacuctl status
```

Shows `kacuctl`'s own state plus live RAM/CPU/load info, read via a small
companion C utility (`kacu-sysinfo/`) that parses `/proc` directly — no
shelling out to `free`/`lscpu`.

## Project layout

```
kacuctl/          kacuctl itself (Rust)
kacu-sysinfo/   small C utility, RAM/CPU/load from /proc, used by `kacuctl status`
```

## License

MIT
