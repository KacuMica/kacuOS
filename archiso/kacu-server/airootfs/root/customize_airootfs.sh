#!/usr/bin/env python3
# ==========================================================================
# customize_airootfs.sh (legacy sway/server profiles, Python)
# ==========================================================================
from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path


def run(cmd: list[str]) -> int:
    print(f"+ {' '.join(cmd)}", flush=True)
    try:
        return subprocess.run(cmd).returncode
    except FileNotFoundError:
        print(f"warn: command not found: {cmd[0]}", file=sys.stderr)
        return 127


def main() -> int:
    setup = Path("/usr/local/bin/kacu-setup")
    if setup.exists():
        run([str(setup), "--system"])
    else:
        run(["systemctl", "enable", "systemd-zram-setup@zram0.service"])

    src = Path("/etc/kacu/kacu-os-release")
    if src.exists():
        dst = Path("/etc/os-release")
        dst.unlink(missing_ok=True)
        shutil.copy2(src, dst)
        print("==> branding: /etc/os-release updated")

    print("==> customize_airootfs.sh done")
    return 0


if __name__ == "__main__":
    sys.exit(main())
