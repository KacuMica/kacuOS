#!/usr/bin/env bash
# shellcheck disable=SC2034
iso_name="kacu-server"
iso_label="KACUOS_$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y%m)"
iso_publisher="kacuOS <https://github.com/yourname/kacu>"
iso_application="kacuOS $([ "kacu-server" = "kacu-desktop" ] && echo Desktop || echo Server) Live/Install"
iso_version="$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y.%m.%d)"
install_dir="arch"
buildmodes=('iso')
bootmodes=('bios.syslinux'
           'uefi.systemd-boot')
pacman_conf="pacman.conf"
airootfs_image_type="squashfs"
airootfs_image_tool_options=('-comp' 'xz' '-Xbcj' 'x86' '-b' '1M' '-Xdict-size' '1M')
bootstrap_tarball_compression=('zstd' '-c' '-T0' '--auto-threads=logical' '--long' '-19')
file_permissions=(
  ["/etc/shadow"]="0:0:400"
  ["/etc/kacu/welcome.py"]="0:0:755"
  ["/etc/profile.d/kacu-welcome.sh"]="0:0:755"
  ["/etc/profile.d/kacu-neofetch.sh"]="0:0:755"
  ["/usr/local/bin/kacuctl"]="0:0:755"
  ["/usr/local/bin/kacu-sysinfo"]="0:0:755"
  ["/usr/local/bin/kacu-setup"]="0:0:755"
  ["/usr/local/bin/kacu-pkg"]="0:0:755"
  ["/usr/local/bin/kacu_install"]="0:0:755"
  ["/usr/local/bin/langley"]="0:0:755"
  ["/etc/kacu/presets/hooks/server-post.py"]="0:0:755"
  ["/etc/kacu/presets/hooks/ai-post.py"]="0:0:755"
  ["/root/customize_airootfs.sh"]="0:0:755"
)
