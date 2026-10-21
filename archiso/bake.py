#!/usr/bin/env python3
# ==========================================================================
# bake.py — build the kacuOS tooling and bake it into archiso profiles.
#
# Files under airootfs/ are copied byte-for-byte into the ISO, so this has to
# run before mkarchiso whenever kacuctl, kacu-sysinfo, kacu-pkg, kacu_install
# or their presets change.
#
# Usage:
#   ./bake.py                      # build everything, bake into every profile
#   ./bake.py --skip-build         # reuse existing binaries (e.g. on Windows)
#   ./bake.py --profile kacu-desktop
# ==========================================================================
from __future__ import annotations

import argparse
import shutil
import stat
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT = SCRIPT_DIR.parent

# Files that must be executable inside the ISO.
EXECUTABLE_NAMES = {
    "kacuctl", "kacu-sysinfo", "kacu-pkg", "kacu_install", "kacu-setup",
}


def run(cmd: list[str], cwd: Path | None = None, dry_run: bool = False) -> int:
    print(f"+ {' '.join(cmd)}", flush=True)
    if dry_run:
        return 0
    return subprocess.call(cmd, cwd=str(cwd) if cwd else None)


def make_executable(path: Path) -> None:
    if not path.exists():
        return
    mode = path.stat().st_mode
    path.chmod(mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)


def copy_file(src: Path, dst: Path, executable: bool = False) -> bool:
    """Copy src to dst; return False (with a warning) when src is missing."""
    if not src.exists():
        print(f"  warn: missing {src.relative_to(ROOT)}, skipping")
        return False
    # kacu-setup is authored inside kacu-desktop; copying it onto itself is a
    # no-op, but shutil would raise SameFileError, so short-circuit it.
    if dst.exists() and src.resolve() == dst.resolve():
        if executable:
            make_executable(dst)
        return True
    dst.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(src, dst)
    if executable or dst.name in EXECUTABLE_NAMES or dst.suffix in {".py", ".sh"}:
        make_executable(dst)
    return True


def build_kacuctl(dry_run: bool) -> Path | None:
    manifest = ROOT / "kacuctl" / "Cargo.toml"
    if not manifest.exists():
        print("  warn: kacuctl/Cargo.toml not found, skipping Rust build")
        return None
    if run(["cargo", "build", "--release"], cwd=manifest.parent, dry_run=dry_run) != 0:
        raise SystemExit("error: cargo build failed")
    return ROOT / "kacuctl" / "target" / "release" / "kacuctl"


def build_kacu_sysinfo(dry_run: bool) -> Path | None:
    makefile = ROOT / "kacu-sysinfo" / "Makefile"
    if not makefile.exists():
        print("  warn: kacu-sysinfo/Makefile not found, skipping C build")
        return None
    if run(["make"], cwd=makefile.parent, dry_run=dry_run) != 0:
        raise SystemExit("error: make failed")
    return ROOT / "kacu-sysinfo" / "kacu-sysinfo"


def discover_profiles(explicit: list[str]) -> list[Path]:
    if explicit:
        profiles = [SCRIPT_DIR / name for name in explicit]
    else:
        profiles = sorted(path.parent for path in SCRIPT_DIR.glob("*/packages.x86_64"))
    missing = [p for p in profiles if not (p / "packages.x86_64").exists()]
    if missing:
        raise SystemExit(f"error: not an archiso profile: {', '.join(map(str, missing))}")
    return profiles


def bake_profile(profile: Path, binaries: dict[str, Path | None]) -> None:
    print(f"\n==> baking into {profile.name}")
    airootfs = profile / "airootfs"

    # 1. Compiled tools.
    for name, binary in binaries.items():
        if binary is None:
            continue
        copy_file(binary, airootfs / "usr/local/bin" / name, executable=True)

    # 2. Python tooling (kacu-pkg, kacu-setup, kacu_install).
    copy_file(ROOT / "kacu-pkg" / "kacu-pkg",
              airootfs / "usr/local/bin/kacu-pkg", executable=True)
    copy_file(ROOT / "kacu-install" / "kacu_install",
              airootfs / "usr/local/bin/kacu_install", executable=True)

    # kacu-setup is authored inside the GNOME profile; share it with the others.
    setup_src = ROOT / "archiso" / "kacu-desktop" / "airootfs" / "usr/local/bin/kacu-setup"
    copy_file(setup_src, airootfs / "usr/local/bin/kacu-setup", executable=True)

    # 3. Presets and the installer's package list.
    presets_src = ROOT / "kacu-pkg" / "presets"
    presets_dst = airootfs / "etc/kacu/presets"
    if presets_src.is_dir():
        if presets_dst.exists():
            shutil.rmtree(presets_dst)
        shutil.copytree(presets_src, presets_dst)
        for hook in presets_dst.rglob("*"):
            if hook.is_file() and hook.suffix in {".py", ".sh"}:
                make_executable(hook)
    else:
        print("  warn: kacu-pkg/presets not found")

    copy_file(ROOT / "kacu-install" / "install-packages.txt",
              airootfs / "etc/kacu/install-packages.txt")


def main() -> int:
    parser = argparse.ArgumentParser(description="Bake kacuOS tooling into archiso profiles")
    parser.add_argument("--profile", action="append", default=[],
                        help="profile directory name (repeatable; default: all)")
    parser.add_argument("--skip-build", action="store_true",
                        help="do not run cargo/make, use existing binaries")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    profiles = discover_profiles(args.profile)
    print(f"profiles: {', '.join(p.name for p in profiles)}")

    binaries: dict[str, Path | None] = {}
    if args.skip_build:
        for name in ("kacuctl", "kacu-sysinfo"):
            candidate = ROOT / ("kacuctl/target/release/kacuctl" if name == "kacuctl"
                                else "kacu-sysinfo/kacu-sysinfo")
            binaries[name] = candidate if candidate.exists() else None
            if binaries[name] is None:
                print(f"  warn: {name} not built and --skip-build was given")
    else:
        binaries["kacuctl"] = build_kacuctl(args.dry_run)
        binaries["kacu-sysinfo"] = build_kacu_sysinfo(args.dry_run)

    for profile in profiles:
        bake_profile(profile, binaries)

    print("\n==> done")
    print("    rebuild with: cd archiso/<profile> && sudo mkarchiso -v -o out/ .")
    return 0


if __name__ == "__main__":
    sys.exit(main())
