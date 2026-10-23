#!/usr/bin/env python3
# ==========================================================================
# kacu-pkg post-install hook for the "server" box.
# Runs after the server packages are installed.
# ==========================================================================
from __future__ import annotations

import os
import shutil
import subprocess
import sys


def sudo() -> list[str]:
    return [] if os.geteuid() == 0 else ["sudo"]


def run(cmd: list[str], shell: bool = False) -> int:
    try:
        return subprocess.call(cmd, shell=shell)
    except FileNotFoundError:
        return 127


def main() -> int:
    print("==> Enabling Docker...")
    if run(sudo() + ["systemctl", "enable", "--now", "docker"]) != 0:
        run(sudo() + ["systemctl", "enable", "docker"])

    # k3s: minimal single-node cluster (no traefik/servicelb/metrics-server).
    if shutil.which("k3s"):
        print("==> k3s already installed, skipping.")
    else:
        print("==> Installing k3s (server, no traefik/servicelb/metrics-server)...")
        command = (
            "curl -sfL https://get.k3s.io | "
            "INSTALL_K3S_EXEC='server --disable traefik --disable servicelb "
            "--disable metrics-server' sh -"
        )
        if run(command, shell=True) != 0:
            print("warn: k3s installation failed (check network access)", file=sys.stderr)

    print("==> Done. Check the node with: sudo k3s kubectl get nodes")
    return 0


if __name__ == "__main__":
    sys.exit(main())
