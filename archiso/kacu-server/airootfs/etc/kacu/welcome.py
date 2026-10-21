#!/usr/bin/env python3
# ==========================================================================
# welcome.py — first-login welcome for the legacy sway/server profiles.
# ==========================================================================
from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path


def run(cmd: list[str], shell: bool = False) -> int:
    try:
        return subprocess.call(cmd, shell=shell)
    except FileNotFoundError:
        return 127


def main() -> int:
    marker = Path.home() / ".kacu-welcomed"
    if marker.exists():
        return 0

    print("\n==================================")
    print("   Welcome to kacuOS")
    print("==================================\n")
    print("Choose a shell to work with:")
    print("  1) bash")
    print("  2) zsh")

    try:
        choice = input("> ").strip()
    except (EOFError, KeyboardInterrupt):
        choice = "1"

    if choice == "2":
        zsh = "/usr/bin/zsh"
        if os.geteuid() == 0:
            run(["chsh", "-s", zsh, os.environ.get("USER", "root")])
        else:
            run(["sudo", "chsh", "-s", zsh, os.environ.get("USER", "")])
        print("zsh selected. This takes effect on next login.\n")
        print("kacuOS recommends Oh My Zsh for themes and plugins:")
        print("  https://ohmyz.sh/\n")
        try:
            answer = input("Install Oh My Zsh now? [y/N] ").strip().lower()
        except (EOFError, KeyboardInterrupt):
            answer = ""
        if answer == "y":
            run(
                'sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended',
                shell=True,
            )
            print("Oh My Zsh installed. Log back in as zsh to see it.")
        else:
            print("Skipped. You can install it later from https://ohmyz.sh/")
    else:
        print("bash selected (default).")

    print("\nRun 'kacu-pkg list' to see available boxes (dev, server, ai).")
    print("Run 'kacu-sysinfo' to see system info.\n")

    marker.touch()
    return 0


if __name__ == "__main__":
    sys.exit(main())
