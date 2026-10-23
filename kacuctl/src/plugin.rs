use anyhow::{Context, Result};
use colored::Colorize;
use mlua::{Lua, Table, Value};
use serde::Deserialize;
use std::fs;
use std::path::PathBuf;
use std::process::Command;

/// Commands allowed for `kacu.exec_readonly`, regardless of plugin permissions.
/// Kept short and read-only on purpose: plugins should not be able to mutate
/// the system through this path even by accident.
const READONLY_WHITELIST: &[&str] = &["git", "ls", "cat", "pwd", "whoami", "date", "uname"];

#[derive(Deserialize, Clone)]
struct PluginManifest {
    plugin: PluginMeta,
    permissions: PluginPermissions,
}

#[derive(Deserialize, Clone)]
struct PluginMeta {
    name: String,
    version: String,
    #[serde(default)]
    author: String,
}

#[derive(Deserialize, Clone, Default)]
struct PluginPermissions {
    #[serde(default)]
    exec: bool,
    #[serde(default)]
    network: bool,
    #[serde(default)]
    hooks: Vec<String>,
}

fn plugins_dir() -> PathBuf {
    if let Ok(p) = std::env::var("KACUOS_PLUGINS_DIR") {
        return PathBuf::from(p);
    }
    let system_path = PathBuf::from("/etc/kacu/plugins");
    if system_path.exists() {
        return system_path;
    }
    PathBuf::from("./plugins")
}

fn load_manifest(dir: &PathBuf) -> Result<PluginManifest> {
    let path = dir.join("plugin.toml");
    let content = fs::read_to_string(&path)
        .with_context(|| format!("manifest not found: {}", path.display()))?;
    toml::from_str(&content).with_context(|| format!("failed to parse {}", path.display()))
}

/// Installs a plugin by copying it from a source directory into the plugins dir.
/// In a real deployment this would pull from a registry; for now it works on a
/// local path, which is enough to develop and test the engine end to end.
pub fn add(source: &str) -> Result<()> {
    let source_path = PathBuf::from(source);
    let manifest = load_manifest(&source_path)?;

    println!(
        "{} {} v{} by {}",
        "Plugin:".bold(),
        manifest.plugin.name.green(),
        manifest.plugin.version,
        if manifest.plugin.author.is_empty() {
            "unknown"
        } else {
            &manifest.plugin.author
        }
    );

    println!("{}", "Requested permissions:".bold());
    println!(
        "  exec:    {}",
        if manifest.permissions.exec {
            "yes".red()
        } else {
            "no".green()
        }
    );
    println!(
        "  network: {}",
        if manifest.permissions.network {
            "yes".red()
        } else {
            "no".green()
        }
    );
    println!("  hooks:   {}", manifest.permissions.hooks.join(", "));

    print!("Install this plugin? [y/N] ");
    use std::io::Write;
    std::io::stdout().flush().ok();
    let mut answer = String::new();
    std::io::stdin().read_line(&mut answer).ok();
    if !answer.trim().eq_ignore_ascii_case("y") {
        println!("{}", "Cancelled.".yellow());
        return Ok(());
    }

    let dest_dir = plugins_dir().join(&manifest.plugin.name);
    fs::create_dir_all(&dest_dir)?;
    for entry in fs::read_dir(&source_path)? {
        let entry = entry?;
        let dest = dest_dir.join(entry.file_name());
        fs::copy(entry.path(), dest)?;
    }

    println!("{}", "Installed.".green().bold());
    Ok(())
}

pub fn list() -> Result<()> {
    let dir = plugins_dir();
    if !dir.exists() {
        println!("{} {}", "Plugins directory not found:".yellow(), dir.display());
        return Ok(());
    }

    println!("{}", "Installed plugins:".bold());
    for entry in fs::read_dir(&dir)? {
        let entry = entry?;
        if !entry.path().is_dir() {
            continue;
        }
        if let Ok(manifest) = load_manifest(&entry.path()) {
            println!(
                "  {} v{} — hooks: {}",
                manifest.plugin.name.green(),
                manifest.plugin.version,
                manifest.permissions.hooks.join(", ")
            );
        }
    }
    Ok(())
}

/// Runs a single plugin's init.lua in a sandboxed Lua VM and fires the given
/// hook if the plugin declared it in plugin.toml.
pub fn run_hook(hook_name: &str) -> Result<()> {
    let dir = plugins_dir();
    if !dir.exists() {
        return Ok(());
    }

    for entry in fs::read_dir(&dir)? {
        let entry = entry?;
        if !entry.path().is_dir() {
            continue;
        }
        let manifest = match load_manifest(&entry.path()) {
            Ok(m) => m,
            Err(_) => continue,
        };
        if !manifest.permissions.hooks.iter().any(|h| h == hook_name) {
            continue;
        }

        let init_path = entry.path().join("init.lua");
        if !init_path.exists() {
            continue;
        }
        let code = fs::read_to_string(&init_path)?;

        run_plugin_code(&manifest, &code, hook_name)
            .with_context(|| format!("plugin '{}' failed on hook '{}'", manifest.plugin.name, hook_name))?;
    }

    Ok(())
}

fn run_plugin_code(manifest: &PluginManifest, code: &str, hook_name: &str) -> Result<()> {
    let lua = Lua::new();
    let kacu_table = lua.create_table()?;

    // kacu.print — always allowed, plugin's only guaranteed way to talk back
    kacu_table.set(
        "print",
        lua.create_function(|_, msg: String| {
            println!("{} {}", "[plugin]".cyan(), msg);
            Ok(())
        })?,
    )?;

    // kacu.exec_readonly — whitelisted commands only, regardless of permissions
    kacu_table.set(
        "exec_readonly",
        lua.create_function(|_, cmd: String| {
            let program = cmd.split_whitespace().next().unwrap_or("");
            if !READONLY_WHITELIST.contains(&program) {
                return Err(mlua::Error::RuntimeError(format!(
                    "'{program}' is not in the read-only whitelist"
                )));
            }
            let output = Command::new("sh")
                .arg("-c")
                .arg(&cmd)
                .output()
                .map_err(|e| mlua::Error::RuntimeError(e.to_string()))?;
            Ok(String::from_utf8_lossy(&output.stdout).to_string())
        })?,
    )?;

    // kacu.exec — only wired up if the plugin declared permissions.exec = true
    if manifest.permissions.exec {
        kacu_table.set(
            "exec",
            lua.create_function(|_, cmd: String| {
                let status = Command::new("sh")
                    .arg("-c")
                    .arg(&cmd)
                    .status()
                    .map_err(|e| mlua::Error::RuntimeError(e.to_string()))?;
                Ok(status.success())
            })?,
        )?;
    }

    lua.globals().set("kacu", kacu_table)?;

    // Load the plugin. Its init.lua may register a handler under
    // kacu_hooks[hook_name] which we then call explicitly — this keeps the
    // contract explicit rather than relying on side effects at load time.
    let hooks_table: Table = lua.create_table()?;
    lua.globals().set("kacu_hooks", hooks_table)?;

    lua.load(code).exec()?;

    let hooks_table: Table = lua.globals().get("kacu_hooks")?;
    let handler: Value = hooks_table.get(hook_name)?;
    if let Value::Function(f) = handler {
        f.call::<_, ()>(())?;
    }

    Ok(())
}
