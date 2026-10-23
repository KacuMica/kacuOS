#!/usr/bin/env python3
# ==========================================================================
# run_qemu.py — launch a kacuOS ISO in QEMU (Windows / Linux / macOS)
#
# Examples (Windows PowerShell):
#   python tools/run_qemu.py --iso C:\isos\kacuos.iso
#   python tools/run_qemu.py --iso C:\isos\kacuos.iso --uefi
#   python tools/run_qemu.py --iso C:\isos\kacuos.iso --disk C:\vm\kacuos.qcow2 --create-disk 20G
#   python tools/run_qemu.py --disk C:\vm\kacuos.qcow2          # boot installed system
#
# Requirements: QEMU for Windows (qemu-system-x86_64.exe + qemu-img.exe).
# ==========================================================================
from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys
from pathlib import Path

WINDOWS_DIRS = [
    Path(r"C:\Program Files\qemu"),
    Path(r"C:\Program Files (x86)\qemu"),
]


def find_binary(name: str) -> str | None:
    """Locate a QEMU binary on PATH or in the usual Windows install dirs."""
    found = shutil.which(name) or shutil.which(name + ".exe")
    if found:
        return found
    for directory in WINDOWS_DIRS:
        candidate = directory / f"{name}.exe"
        if candidate.exists():
            return str(candidate)
    return None


def find_ovmf() -> tuple[Path, Path] | None:
    """Return (code, vars) firmware files for UEFI, if present."""
    search_dirs = [Path(r"C:\Program Files\qemu\share"), Path(r"C:\Program Files\qemu")]
    for directory in search_dirs:
        code = directory / "edk2-x86_64-code.fd"
        vars_file = directory / "edk2-i386-vars.fd"
        if code.exists() and vars_file.exists():
            return code, vars_file
    for code in search_dirs[0].glob("*x86_64*code*.fd") if search_dirs[0].is_dir() else []:
        for vars_file in code.parent.glob("*vars*.fd"):
            return code, vars_file
    return None


def accel_choices(qemu: str) -> set[str]:
    try:
        output = subprocess.run(
            [qemu, "-accel", "help"], text=True,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
        ).stdout.lower()
    except OSError:
        return set()
    return {name for name in ("whpx", "tcg", "kvm", "hvf") if name in output}


def main() -> int:
    parser = argparse.ArgumentParser(description="Launch kacuOS in QEMU")
    parser.add_argument("--iso", help="path to the kacuos ISO")
    parser.add_argument("--disk", help="path to a qcow2/raw disk image")
    parser.add_argument("--create-disk", metavar="SIZE",
                        help="create the disk if missing, e.g. 20G")
    parser.add_argument("--memory", type=int, default=4096, help="RAM in MiB (default 4096)")
    parser.add_argument("--cpus", type=int, default=2, help="vCPU count (default 2)")
    parser.add_argument("--uefi", action="store_true", help="boot with UEFI/OVMF")
    parser.add_argument("--accel", choices=("auto", "whpx", "tcg"), default="auto")
    parser.add_argument("--display", default="default",
                        help="gtk, sdl, default or none (default: let QEMU choose)")
    parser.add_argument("--snapshot", action="store_true", help="do not persist disk writes")
    parser.add_argument("--no-net", action="store_true", help="disable user-mode networking")
    parser.add_argument("--extra", default="", help="extra QEMU args (quoted string)")
    args = parser.parse_args()

    qemu = find_binary("qemu-system-x86_64")
    if not qemu:
        print("error: qemu-system-x86_64 not found. Install QEMU for Windows and add it to PATH.",
              file=sys.stderr)
        return 1
    if not args.iso and not args.disk:
        print("error: give --iso, --disk, or both", file=sys.stderr)
        return 1

    if args.iso and not Path(args.iso).exists():
        print(f"error: ISO not found: {args.iso}", file=sys.stderr)
        return 1

    # Create the disk image if requested.
    if args.create_disk and args.disk and not Path(args.disk).exists():
        qemu_img = find_binary("qemu-img")
        if not qemu_img:
            print("error: qemu-img not found", file=sys.stderr)
            return 1
        Path(args.disk).parent.mkdir(parents=True, exist_ok=True)
        print(f"+ qemu-img create -f qcow2 {args.disk} {args.create_disk}")
        subprocess.run([qemu_img, "create", "-f", "qcow2", args.disk, args.create_disk],
                       check=True)

    accels = accel_choices(qemu)
    if args.accel == "auto":
        accel = "whpx" if "whpx" in accels else ("kvm" if "kvm" in accels else "tcg")
    else:
        accel = args.accel
        if accel not in accels:
            print(f"warning: '{accel}' is not reported by this QEMU build; trying anyway",
                  file=sys.stderr)

    cmd = [
        qemu,
        "-machine", "q35",
        "-m", str(args.memory),
        "-smp", str(args.cpus),
        "-accel", accel,
        "-device", "virtio-vga",
        "-usb", "-device", "usb-tablet",
        "-rtc", "base=localtime",
        "-display", args.display,
    ]

    if not args.no_net:
        cmd += ["-nic", "user,model=virtio-net-pci"]

    if args.uefi:
        firmware = find_ovmf()
        if not firmware:
            print("error: OVMF files not found (edk2-x86_64-code.fd / edk2-i386-vars.fd)",
                  file=sys.stderr)
            return 1
        code, vars_template = firmware
        vars_copy = Path.cwd() / "kacuos-vars.fd"
        if not vars_copy.exists():
            shutil.copy2(vars_template, vars_copy)
        cmd += [
            "-drive", f"if=pflash,format=raw,readonly=on,file={code}",
            "-drive", f"if=pflash,format=raw,file={vars_copy}",
        ]

    if args.disk:
        if not Path(args.disk).exists():
            print(f"error: disk not found: {args.disk} (use --create-disk)", file=sys.stderr)
            return 1
        cmd += ["-drive", f"file={args.disk},if=virtio,format=qcow2"]
    if args.iso:
        cmd += ["-cdrom", str(Path(args.iso).resolve()), "-boot", "d"]
    elif args.disk:
        cmd += ["-boot", "c"]

    if args.snapshot:
        cmd += ["-snapshot"]
    if args.extra:
        cmd += args.extra.split()

    print("+ " + " ".join(f'"{part}"' if " " in part else part for part in cmd))
    try:
        return subprocess.call(cmd)
    except KeyboardInterrupt:
        return 130


if __name__ == "__main__":
    sys.exit(main())
