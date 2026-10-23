use anyhow::{Context, Result};
use colored::Colorize;
use serde::Deserialize;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;

#[derive(Deserialize)]
struct PresetFile {
    preset: PresetMeta,
    packages: PresetPackages,
    hooks: Option<PresetHooks>,
}

#[derive(Deserialize)]
struct PresetMeta {
    name: String,
    description: String,
    #[allow(dead_code)]
    profile: Vec<String>,
}

#[derive(Deserialize)]
struct PresetPackages {
    #[serde(default)]
    pacman: Vec<String>,
}

#[derive(Deserialize)]
struct PresetHooks {
    post_install: Option<String>,
}

/// Directory containing presets. On a real system this is /etc/kacu/presets,
/// but for local development without an installed system we look next to the binary.
fn presets_dir() -> PathBuf {
    if let Ok(p) = std::env::var("KACUOS_PRESETS_DIR") {
        return PathBuf::from(p);
    }
    let system_path = PathBuf::from("/etc/kacu/presets");
    if system_path.exists() {
        return system_path;
    }
    PathBuf::from("./presets")
}

fn load_preset(name: &str) -> Result<PresetFile> {
    let path = presets_dir().join(format!("{name}.toml"));
    let content = fs::read_to_string(&path)
        .with_context(|| format!("preset file not found: {}", path.display()))?;
    let parsed: PresetFile = toml::from_str(&content)
        .with_context(|| format!("failed to parse {}", path.display()))?;
    Ok(parsed)
}

pub fn install(name: &str, dry_run: bool) -> Result<()> {
    let preset = load_preset(name)?;

    println!(
        "{} {} — {}",
        "Preset:".bold(),
        preset.preset.name.green(),
        preset.preset.description
    );

    if preset.packages.pacman.is_empty() {
        println!("{}", "No pacman packages in this preset.".yellow());
    } else {
        println!(
            "{} {}",
            "Packages:".bold(),
            preset.packages.pacman.join(", ")
        );

        let mut cmd_args = vec!["-Sy".to_string(), "--noconfirm".to_string()];
        cmd_args.extend(preset.packages.pacman.iter().cloned());

        if dry_run {
            println!(
                "{} sudo pacman {}",
                "[dry-run] would run:".cyan(),
                cmd_args.join(" ")
            );
        } else {
            let status = Command::new("sudo")
                .arg("pacman")
                .args(&cmd_args)
                .status()
                .context("failed to run pacman")?;

            if !status.success() {
                anyhow::bail!("pacman exited with an error");
            }
        }
    }

    if let Some(hooks) = &preset.hooks {
        if let Some(script) = &hooks.post_install {
            let script_path = presets_dir().join(script);
            if dry_run {
                println!(
                    "{} {}",
                    "[dry-run] would run post-install hook:".cyan(),
                    script_path.display()
                );
            } else if script_path.exists() {
                println!("{} {}", "Running hook:".bold(), script_path.display());
                // Pick the interpreter from the extension: .py -> python3,
                // .lua -> lua, anything else -> bash (kept for old presets).
                let extension = script_path
                    .extension()
                    .and_then(|s| s.to_str())
                    .unwrap_or("");
                let (program, args): (&str, Vec<String>) = match extension {
                    "py" => ("python3", vec![script_path.display().to_string()]),
                    "lua" => ("lua", vec![script_path.display().to_string()]),
                    _ => ("bash", vec![script_path.display().to_string()]),
                };
                let status = Command::new(program)
                    .args(&args)
                    .status()
                    .context("failed to run post-install hook")?;
                if !status.success() {
                    anyhow::bail!("post-install hook exited with an error");
                }
            } else {
                println!(
                    "{} {}",
                    "warn: hook file not found:".yellow(),
                    script_path.display()
                );
            }
        }
    }

    println!("{}", "Done.".green().bold());
    Ok(())
}

pub fn list() -> Result<()> {
    let dir = presets_dir();
    if !dir.exists() {
        println!(
            "{} {}",
            "Presets directory not found:".yellow(),
            dir.display()
        );
        return Ok(());
    }

    println!("{}", "Available presets:".bold());
    for entry in fs::read_dir(&dir)? {
        let entry = entry?;
        let path = entry.path();
        if path.extension().and_then(|s| s.to_str()) == Some("toml") {
            if let Ok(preset) = load_preset(
                path.file_stem()
                    .and_then(|s| s.to_str())
                    .unwrap_or_default(),
            ) {
                println!(
                    "  {} — {}",
                    preset.preset.name.green(),
                    preset.preset.description
                );
            }
        }
    }
    Ok(())
}

pub fn status() -> Result<()> {
    println!("{}", "kacuOS status".bold());
    println!("kacuctl version: {}", env!("CARGO_PKG_VERSION"));
    println!("presets dir: {}", presets_dir().display());
    println!();

    // kacu-sysinfo is a separate C binary (see kacu-sysinfo/), invoked as a
    // subprocess. It stays a standalone tool rather than logic embedded in
    // kacuctl itself: it has no reason to touch anything beyond /proc, so it
    // gets no dependencies, no Lua sandbox, no permission model — just a
    // small, fast, static binary that reads and prints.
    match Command::new("kacu-sysinfo").output() {
        Ok(output) if output.status.success() => {
            print!("{}", String::from_utf8_lossy(&output.stdout));
        }
        Ok(_) => {
            println!(
                "{}",
                "warn: kacu-sysinfo exited with an error".yellow()
            );
        }
        Err(_) => {
            println!(
                "{}",
                "warn: kacu-sysinfo not found in PATH (not installed?)".yellow()
            );
        }
    }

    Ok(())
}

#[allow(dead_code)]
fn is_installed(_pkg: &str) -> bool {
    // TODO: check via `pacman -Q` for kacuctl status
    false
}

#[allow(dead_code)]
fn presets_exists(dir: &Path) -> bool {
    dir.exists()
}
