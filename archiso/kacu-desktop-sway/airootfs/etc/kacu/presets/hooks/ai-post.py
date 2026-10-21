#!/usr/bin/env python3
# ==========================================================================
# kacu-pkg post-install hook for the "ai" box.
# Ollama is in the official repos; llama.cpp is AUR-only, so try pacman, then
# an AUR helper, then print manual build instructions.
# ==========================================================================
from __future__ import annotations

import os
import shutil
import subprocess
import sys


def sudo() -> list[str]:
    return [] if os.geteuid() == 0 else ["sudo"]


def run(cmd: list[str]) -> int:
    try:
        return subprocess.call(cmd)
    except FileNotFoundError:
        return 127


def main() -> int:
    print("==> Enabling the Ollama service...")
    if run(sudo() + ["systemctl", "enable", "--now", "ollama"]) != 0:
        run(sudo() + ["systemctl", "enable", "ollama"])
    print("    Pull a model later with:  ollama pull llama3.2")

    print("==> Looking for llama.cpp...")
    if shutil.which("llama-cli") or shutil.which("llama-server"):
        print("    llama.cpp already installed, skipping.")
    elif run(["pacman", "-Si", "llama.cpp"]) == 0:
        run(sudo() + ["pacman", "-S", "--needed", "--noconfirm", "llama.cpp"])
    elif shutil.which("paru"):
        run(["paru", "-S", "--needed", "--noconfirm", "llama-cpp"])
    elif shutil.which("yay"):
        run(["yay", "-S", "--needed", "--noconfirm", "llama-cpp"])
    else:
        print("    llama.cpp is not in the official repos and no AUR helper is present.")
        print("    Build it manually:")
        print("      git clone https://github.com/ggml-org/llama.cpp")
        print("      cmake -B build llama.cpp && cmake --build build -j\"$(nproc)\"")

    print("==> Done.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
