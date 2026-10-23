#!/usr/bin/env bash
# kacuOS archiso profile definition (GNOME desktop, live/install ISO).
# shellcheck disable=SC2034
iso_name="kacuos"
iso_label="KACUOS_$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y%m)"
iso_publisher="kacuOS <https://github.com/kacu-os/kacuOS>"
iso_application="kacuOS Live/Install (minimal GNOME)"
iso_version="$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y.%m.%d)"
install_dir="arch"
buildmodes=('iso')
bootmodes=('bios.syslinux'
           'uefi.systemd-boot')
pacman_conf="pacman.conf"
airootfs_image_type="squashfs"
# xz + x86 BCJ filter gives a good size/speed tradeoff for an x86_64-only image.
airootfs_image_tool_options=('-comp' 'xz' '-Xbcj' 'x86' '-b' '1M' '-Xdict-size' '1M')
bootstrap_tarball_compression=('zstd' '-c' '-T0' '--auto-threads=logical' '--long' '-19')
# Executable bits are NOT inferred automatically — list every script that
# must be runnable in the live system here.
file_permissions=(
  ["/root/customize_airootfs.sh"]="0:0:755"
<<<<<<< HEAD
  ["/usr/local/bin/kacuctl"]="0:0:755"
  ["/usr/local/bin/kacu-sysinfo"]="0:0:755"
  ["/usr/local/bin/kacu-setup"]="0:0:755"
  ["/usr/local/bin/kacu-pkg"]="0:0:755"
  ["/usr/local/bin/kacu_install"]="0:0:755"
  ["/usr/local/bin/langley"]="0:0:755"
=======
  ["/usr/local/bin/kacu-setup"]="0:0:755"
  ["/usr/local/bin/kacu-pkg"]="0:0:755"
  ["/usr/local/bin/kacu_install"]="0:0:755"
>>>>>>> 8d49f839a36a99c78b1b0c3a21e06879348582fa
  ["/etc/kacu/presets/hooks/server-post.py"]="0:0:755"
  ["/etc/kacu/presets/hooks/ai-post.py"]="0:0:755"
)
