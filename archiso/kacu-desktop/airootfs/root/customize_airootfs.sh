#!/usr/bin/env python3
# ==========================================================================
# customize_airootfs.sh (Python)
#
# Runs inside the chroot during mkarchiso, after packages are installed but
# before the squashfs image is created. Everything here is baked into the ISO.
#
# NOTE: the filename is fixed by archiso; the executable bit + shebang decide
# the interpreter, so this file is Python despite the .sh suffix.
# ==========================================================================
from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path


def run(cmd: list[str], check: bool = False) -> int:
    print(f"+ {' '.join(cmd)}", flush=True)
    try:
        return subprocess.run(cmd).returncode
    except FileNotFoundError:
        print(f"warn: command not found: {cmd[0]}", file=sys.stderr)
        return 127


def main() -> int:
    # 1. System tuning: zram, dconf defaults, masked services, enabled units.
    setup = Path("/usr/local/bin/kacu-setup")
    if setup.exists():
        run([str(setup), "--system"], check=True)
    else:
        print("warn: /usr/local/bin/kacu-setup missing", file=sys.stderr)

    # 2. Branding (os-release is a symlink on Arch, so replace it).
    print("==> kacuOS: writing /etc/os-release")
    # Live ISO only: passwordless root on the console (liveuser still has sudo).
    run(["passwd", "-d", "root"])
    os_release = Path("/etc/os-release")
    os_release.unlink(missing_ok=True)
    os_release.write_text(
        'NAME="kacuOS"\n'
        'PRETTY_NAME="kacuOS (rolling, GNOME)"\n'
        "ID=kacuos\n"
        "ID_LIKE=arch\n"
        "BUILD_ID=rolling\n"
        'ANSI_COLOR="38;2;23;147;209"\n'
        'HOME_URL="https://github.com/kacu-os/kacuOS"\n'
        'LOGO=kacuos\n',
        encoding="utf-8",
    )

    # 3. Live user. GDM refuses a Wayland session for root, so the live
    #    environment needs a normal user with passwordless sudo.
    print("==> kacuOS: creating the live user")
    if subprocess.run(["id", "liveuser"], stdout=subprocess.DEVNULL,
                      stderr=subprocess.DEVNULL).returncode != 0:
        run(["useradd", "-m", "-G", "wheel,video,audio,storage,network",
             "-s", "/bin/bash", "liveuser"])
        run(["passwd", "-d", "liveuser"])
        sudoers = Path("/etc/sudoers.d/10-liveuser")
        sudoers.parent.mkdir(parents=True, exist_ok=True)
        sudoers.write_text("liveuser ALL=(ALL) NOPASSWD: ALL\n", encoding="utf-8")
        sudoers.chmod(0o440)

    # 4. GDM: Wayland + autologin for the live session.
    print("==> kacuOS: configuring GDM")
    gdm_conf = Path("/etc/gdm/custom.conf")
    gdm_conf.parent.mkdir(parents=True, exist_ok=True)
    gdm_conf.write_text(
        "[daemon]\n"
        "WaylandEnable=true\n"
        "AutomaticLoginEnable=True\n"
        "AutomaticLogin=liveuser\n",
        encoding="utf-8",
    )

    # 5. Enable the display manager and networking.
    run(["systemctl", "enable", "gdm.service"])
    run(["systemctl", "enable", "NetworkManager.service"])
    run(["systemctl", "set-default", "graphical.target"])

    print("==> customize_airootfs.sh done")
    return 0


if __name__ == "__main__":
    sys.exit(main())
