#!/usr/bin/env python3
# ==========================================================================
# kacuctl post-install hook for the "editor" preset (VS Code tarball).
# Arch repos do not ship VS Code; download the official tarball into /opt.
# ==========================================================================
from __future__ import annotations

import os
import shutil
import subprocess
import sys
import tarfile
import tempfile
import urllib.request
from pathlib import Path

URL = "https://code.visualstudio.com/sha/download?build=stable&os=linux-x64"


def sudo() -> list[str]:
    return [] if os.geteuid() == 0 else ["sudo"]


def run(cmd: list[str]) -> int:
    try:
        return subprocess.call(cmd)
    except FileNotFoundError:
        return 127


def main() -> int:
    print("Downloading VS Code (official tarball)...")
    tmp = Path(tempfile.gettempdir()) / "vscode.tar.gz"
    try:
        urllib.request.urlretrieve(URL, tmp)
    except Exception as exc:  # noqa: BLE001 - report and stop cleanly
        print(f"error: download failed: {exc}", file=sys.stderr)
        return 1

    print("Installing to /opt/vscode...")
    run(sudo() + ["mkdir", "-p", "/opt/vscode"])
    # Use tarfile so we do not depend on the `tar` binary.
    with tarfile.open(tmp, "r:gz") as archive:
        members = []
        for member in archive.getmembers():
            parts = Path(member.name).parts
            if len(parts) <= 1:
                continue
            member.name = str(Path(*parts[1:]))  # drop the top-level directory
            members.append(member)
        archive.extractall("/tmp/vscode-extract", members=members)
    run(sudo() + ["cp", "-a", "/tmp/vscode-extract/.", "/opt/vscode/"])
    run(sudo() + ["ln", "-sf", "/opt/vscode/bin/code", "/usr/local/bin/code"])
    shutil.rmtree("/tmp/vscode-extract", ignore_errors=True)
    tmp.unlink(missing_ok=True)

    print("VS Code installed. Run: code")
    return 0


if __name__ == "__main__":
    sys.exit(main())
