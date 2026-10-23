# kacu-pkg

`kacu-pkg` is the package **"box" (preset) manager** for kacuOS. It replaces
the preset half of the old Rust `kacuctl` tool with a single, dependency-free
Python script.

## Why Python

* Python 3.11+ ships in the ISO already and includes `tomllib` in the standard
  library — no build step, no third-party dependencies, no vendored crates.
* The whole tool is one file; it can be baked straight into `airootfs` and
  edited on the live system.
* `kacuctl` (Rust) can stay around for its Lua **plugin** sandbox, but preset
  handling no longer needs a compiled binary.

## Layout

```
kacu-pkg/
├── kacu-pkg                 # the executable (python3, no extension)
└── presets/
    ├── dev.toml             # developer box
    ├── server.toml          # docker + k3s box
    ├── ai.toml              # ollama + llama.cpp box
    └── hooks/
        ├── server-post.sh
        └── ai-post.sh
```

On the built system these land in:

* `/usr/local/bin/kacu-pkg`
* `/etc/kacu/presets/**`

## Preset format (TOML)

```toml
[preset]
name = "dev"
description = "…"
profile = ["desktop"]        # optional, informational

[packages]
pacman = ["git", "neovim"]   # installed with `pacman -S --needed`

[hooks]
pre_install  = "hooks/x.py"  # optional; .py runs via python3
post_install = "hooks/x.py"  # optional; .lua runs via the lua interpreter,
                             # .sh via bash (legacy compatibility)
```

## Usage

```bash
kacu-pkg list
kacu-pkg info ai
kacu-pkg install dev
kacu-pkg install server ai
kacu-pkg install --all --dry-run
kacu-pkg status
```

`KACU_PRESETS_DIR=/some/dir kacu-pkg list` picks up presets from elsewhere,
which is handy for development without an installed system.
