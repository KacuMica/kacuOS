# Writing kacuctl plugins

A plugin is a directory with two files:

```
my-plugin/
  plugin.toml   # metadata + permissions
  init.lua      # code
```

## plugin.toml

```toml
[plugin]
name = "my-plugin"
version = "0.1.0"
author = "your name"

[permissions]
exec = false        # can this plugin run arbitrary shell commands?
network = false      # reserved for future use
hooks = ["shell_prompt"]
```

`hooks` lists which events this plugin responds to. Currently supported:

- `shell_prompt` — fired manually via `kacuctl plugin run-hook shell_prompt`
  (intended to be wired into your shell's prompt command in a future
  version; for now it's a good hook to test plugins against)

## init.lua

Your plugin registers a function per hook it declared:

```lua
kacu_hooks["shell_prompt"] = function()
    kacu.print("hello from my plugin")
end
```

## The `kacu` API

| Function | Requires `permissions.exec` | Description |
|---|---|---|
| `kacu.print(text)` | no | Print a line, prefixed `[plugin]` |
| `kacu.exec_readonly(cmd)` | no | Run a command, but **only** if the first word is on the built-in whitelist: `git`, `ls`, `cat`, `pwd`, `whoami`, `date`, `uname`. Anything else raises a Lua error. Returns stdout as a string. |
| `kacu.exec(cmd)` | **yes** | Run any shell command. Only registered at all if the plugin's manifest sets `permissions.exec = true` — if it's not set, calling `kacu.exec` fails because the function doesn't exist in that plugin's environment. |

This split exists on purpose: most plugins (status indicators, prompt
decorations, read-only info) never need `exec`, and `exec_readonly`'s
whitelist covers the common read-only cases already. Only ask for
`permissions.exec = true` if your plugin genuinely needs to change
something on the system — and expect the user to see that request and
think about it before installing, because `kacuctl plugin add` shows
permissions and asks for confirmation before copying anything in.

## Testing your plugin locally

```bash
export KACUOS_PLUGINS_DIR=./test-plugins
mkdir -p "$KACUOS_PLUGINS_DIR"
kacuctl plugin add ./my-plugin
kacuctl plugin list
kacuctl plugin run-hook shell_prompt
```

## Example

See [`plugins-example/git-status-prompt`](./plugins-example/git-status-prompt)
for a complete working example — prints the current git branch using
`exec_readonly`, no elevated permissions needed.

## What NOT to do

- Don't ask for `permissions.exec = true` if `exec_readonly`'s whitelist
  already covers what you need — it won't be granted by design, and users
  are right to be suspicious of plugins that ask for more than they use.
- Don't try to read files outside what your hook logically needs. There's
  no filesystem sandbox beyond what `exec`/`exec_readonly` gate — be a
  good citizen, the permission model only works if plugin authors respect
  what they declare.
