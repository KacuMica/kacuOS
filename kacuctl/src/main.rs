mod plugin;
mod preset;

use clap::{Parser, Subcommand};
use colored::Colorize;

#[derive(Parser)]
#[command(name = "kacuctl")]
#[command(about = "kacuOS customization CLI", long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Install a preset (e.g. devtools, browser, cluster, media)
    Install {
        preset: String,
        /// Show the commands that would run, without actually installing
        #[arg(long)]
        dry_run: bool,
    },
    /// List available presets
    List,
    /// Show system status (installed presets)
    Status,
    /// Manage Lua plugins
    Plugin {
        #[command(subcommand)]
        action: PluginAction,
    },
}

#[derive(Subcommand)]
enum PluginAction {
    /// Install a plugin from a local directory (shows requested permissions first)
    Add { source: String },
    /// List installed plugins
    List,
    /// Manually fire a hook (mainly useful for testing plugins)
    RunHook { name: String },
}

fn main() {
    let cli = Cli::parse();

    let result = match &cli.command {
        Commands::Install { preset, dry_run } => preset::install(preset, *dry_run),
        Commands::List => preset::list(),
        Commands::Status => preset::status(),
        Commands::Plugin { action } => match action {
            PluginAction::Add { source } => plugin::add(source),
            PluginAction::List => plugin::list(),
            PluginAction::RunHook { name } => plugin::run_hook(name),
        },
    };

    if let Err(e) = result {
        eprintln!("{} {}", "error:".red().bold(), e);
        std::process::exit(1);
    }
}
