// install.go — Step 8: package list, pacstrap, system configuration, GRUB,
// AUR packages (Brave via paru) and kacu-pkg boxes.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	packageListPath = "/etc/kacu/install-packages.txt"
	liveConfigDir   = "/etc/kacu"
	pacmanConfTmp   = "/tmp/kacu-pacman.conf"
)

// builtinPackages is the fallback when /etc/kacu/install-packages.txt is not
// baked into the ISO (mirrors the Python installer's DEFAULT_PACKAGES).
var builtinPackages = []string{
	"base", "linux", "linux-firmware", "intel-ucode", "amd-ucode",
	"sudo", "networkmanager", "wpa_supplicant", "wireless-regdb",
	"grub", "efibootmgr", "os-prober", "e2fsprogs", "dosfstools",
	"zram-generator", "git", "python", "lua", "neovim", "micro",
	"gcc", "make", "pkgconf", "cmake",
	"gnome-shell", "mutter", "gnome-session", "gnome-settings-daemon",
	"gnome-control-center", "gnome-keyring", "gnome-console", "nautilus",
	"gvfs", "xdg-user-dirs", "xdg-desktop-portal-gnome", "xdg-utils",
	"adwaita-icon-theme", "gnome-themes-extra", "gdm", "polkit",
	"accountsservice", "upower", "power-profiles-daemon",
	"pipewire", "pipewire-pulse", "pipewire-alsa", "wireplumber",
	"mesa", "xorg-xwayland", "noto-fonts", "noto-fonts-emoji",
	"ttf-dejavu", "ttf-liberation", "alsa-utils", "sof-firmware",
	"man-db", "less", "bash-completion",
}

// runInstall drives the whole on-disk sequence and always unmounts at the end.
func runInstall(cfg *Config, rootPassword, userPassword string) {
	defer unmountAll()
	if err := installSequence(cfg, rootPassword, userPassword); err != nil {
		warnf("installation aborted; target may be partially configured (%v)", err)
		return
	}
	plog("\n==> Installation complete.")
	plog("    Remove the ISO and reboot:  reboot")
}

// installSequence is the linear pipeline; the first failure aborts it.
func installSequence(cfg *Config, rootPassword, userPassword string) error {
	if err := partitionDisk(cfg.Disk, cfg.UEFI); err != nil {
		return err
	}
	if err := formatAndMount(cfg.Disk, cfg.UEFI); err != nil {
		return err
	}
	if err := installAllPackages(cfg); err != nil {
		return err
	}
	if err := copyKacuosConfig(); err != nil {
		return err
	}
	if err := configureSystem(cfg, rootPassword, userPassword); err != nil {
		return err
	}
	if cfg.AURNeeded {
		installAUR(cfg) // best effort: warns instead of aborting
	}
	installBoxes(cfg.Boxes) // best effort
	return nil
}

// --- package list ----------------------------------------------------------

// loadPackageList reads /etc/kacu/install-packages.txt, falling back to the
// built-in default list.
func loadPackageList() []string {
	data, err := os.ReadFile(packageListPath)
	if err != nil {
		warnf("%s not found, using the built-in package list", packageListPath)
		return append([]string(nil), builtinPackages...)
	}
	var packages []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line != "" {
			packages = append(packages, line)
		}
	}
	if len(packages) == 0 {
		return append([]string(nil), builtinPackages...)
	}
	return packages
}

// buildPackages starts from the base list and layers the user's choices on
// top: audio stack, GPU drivers, microcode, browser, Steam and extra apps.
func buildPackages(cfg *Config) []string {
	packages := applyAudio(loadPackageList(), audioOptionByID(cfg.AudioID))
	packages = append(packages, cfg.GPU...)
	packages = append(packages, cfg.BrowserRepo)
	packages = append(packages, cfg.Apps...)
	if cfg.Steam {
		packages = append(packages, "steam")
		packages = append(packages, steamLib32(cfg.GPUIDs)...)
	}
	if cfg.AURNeeded {
		// paru build environment: group + git for AUR clones.
		packages = append(packages, "base-devel", "git")
	}
	packages = ensureMicrocode(packages)
	return dedupe(packages)
}

// applyAudio removes packages conflicting with the chosen audio stack and
// adds the ones it needs.
func applyAudio(packages []string, option audioOption) []string {
	var result []string
	for _, pkg := range packages {
		conflicts := false
		for _, prefix := range option.remove {
			if strings.HasPrefix(pkg, prefix) {
				conflicts = true
				break
			}
		}
		if !conflicts {
			result = append(result, pkg)
		}
	}
	result = append(result, option.add...)
	return result
}

// steamLib32 adds the 32-bit graphics companions Steam needs (multilib).
func steamLib32(driverIDs []string) []string {
	var pkgs []string
	for _, id := range driverIDs {
		switch id {
		case "intel":
			pkgs = append(pkgs, "lib32-mesa", "lib32-vulkan-intel")
		case "amd":
			pkgs = append(pkgs, "lib32-mesa", "lib32-vulkan-radeon")
		case "nvidia", "nvidia-dkms":
			pkgs = append(pkgs, "lib32-nvidia-utils")
		}
	}
	return pkgs
}

// ensureMicrocode adds the microcode blob matching the CPU vendor.
func ensureMicrocode(packages []string) []string {
	required := ""
	switch {
	case strings.Contains(cpuVendor(), "Intel"):
		required = "intel-ucode"
	case strings.Contains(cpuVendor(), "AMD"):
		required = "amd-ucode"
	}
	if required == "" {
		return packages
	}
	for _, pkg := range packages {
		if pkg == required {
			return packages
		}
	}
	return append(packages, required)
}

// dedupe removes repeated names, keeping first occurrences in order.
func dedupe(items []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

// --- pacstrap ---------------------------------------------------------------

// pacmanConfTemplate is used when the live /etc/pacman.conf is unavailable.
const pacmanConfTemplate = `#
# /etc/pacman.conf — kacuOS (generated by langley)
#
[options]
HoldPkg     = pacman glibc
Architecture = auto
Color
CheckSpace
ParallelDownloads = 5
SigLevel    = Required DatabaseOptional
LocalFileSigLevel = Optional

[core]
Include = /etc/pacman.d/mirrorlist

[extra]
Include = /etc/pacman.d/mirrorlist

#[multilib]
#Include = /etc/pacman.d/mirrorlist
`

// installAllPackages builds the pacman.conf, pacstraps the system, deploys
// the config into the target and writes fstab.
func installAllPackages(cfg *Config) error {
	conf := buildPacmanConf(cfg)
	if err := os.WriteFile(pacmanConfTmp, []byte(conf), 0o644); err != nil {
		return err
	}

	packages := buildPackages(cfg)
	plog("\n==> Installing %d packages (this takes a while)", len(packages))
	cmd := append([]string{"pacstrap", "-C", pacmanConfTmp, "-K", MNTPath}, packages...)
	if err := runStream(cmd, "", ""); err != nil {
		return fmt.Errorf("pacstrap: %w", err)
	}

	// The conf that pacstrap -C used must live inside the target too, or
	// multilib disappears on the first reboot.
	if err := os.WriteFile(MNTPath+"/etc/pacman.conf", []byte(conf), 0o644); err != nil {
		return fmt.Errorf("writing /etc/pacman.conf in target: %w", err)
	}
	if mirrorlist, err := os.ReadFile("/etc/pacman.d/mirrorlist"); err == nil {
		if err := writeTargetFile("etc/pacman.d/mirrorlist", string(mirrorlist), 0o644); err != nil {
			return fmt.Errorf("writing target mirrorlist: %w", err)
		}
	} else {
		warnf("no live /etc/pacman.d/mirrorlist; run mirrorlist setup on first boot")
	}

	return writeFstab()
}

// buildPacmanConf starts from the live conf (or the template) and enables
// multilib when Steam was selected.
func buildPacmanConf(cfg *Config) string {
	conf := pacmanConfTemplate
	if data, err := os.ReadFile("/etc/pacman.conf"); err == nil {
		conf = string(data)
	}
	if cfg.Steam {
		conf = enableMultilib(conf)
	}
	return conf
}

// enableMultilib uncomments only the [multilib] section and its Include line.
func enableMultilib(conf string) string {
	lines := strings.Split(conf, "\n")
	inMultilib := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "#[multilib]":
			lines[i] = "[multilib]"
			inMultilib = true
			continue
		case "[multilib]":
			inMultilib = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inMultilib = false
		}
		if inMultilib && trimmed == "#Include = /etc/pacman.d/mirrorlist" {
			lines[i] = "Include = /etc/pacman.d/mirrorlist"
		}
	}
	return strings.Join(lines, "\n")
}

// writeFstab generates the target fstab from the current mounts.
func writeFstab() error {
	plog("==> Writing /etc/fstab")
	out, err := runCapture([]string{"genfstab", "-U", MNTPath}, "")
	if err != nil {
		return fmt.Errorf("genfstab: %w", err)
	}
	if err := os.WriteFile(MNTPath+"/etc/fstab", []byte(out), 0o644); err != nil {
		return fmt.Errorf("writing fstab: %w", err)
	}
	return nil
}

// --- kacuOS config ----------------------------------------------------------

// copyKacuosConfig copies /etc/kacu, the zram/pacman-side config files and the
// kacu-setup / kacu-pkg binaries into the new system (Python parity).
func copyKacuosConfig() error {
	plog("\n==> Copying kacuOS configuration into the target")
	if info, err := os.Stat(liveConfigDir); err == nil && info.IsDir() {
		target := MNTPath + "/etc/kacu"
		if err := os.MkdirAll(target, 0o755); err != nil {
			return err
		}
		entries, err := os.ReadDir(liveConfigDir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			source := filepath.Join(liveConfigDir, entry.Name())
			destination := filepath.Join(target, entry.Name())
			if entry.IsDir() {
				if err := copyTree(source, destination); err != nil {
					return err
				}
				continue
			}
			if err := copyFile(source, destination, 0o644); err != nil {
				return err
			}
		}
	}

	for _, relative := range []string{
		"etc/systemd/zram-generator.conf",
		"etc/sysctl.d/99-zram.conf",
	} {
		source := filepath.Join("/", relative)
		if _, err := os.Stat(source); err == nil {
			if err := copyFile(source, MNTPath+"/"+relative, 0o644); err != nil {
				return err
			}
		}
	}

	for _, binary := range []string{"kacu-setup", "kacu-pkg"} {
		source := "/usr/local/bin/" + binary
		if _, err := os.Stat(source); err != nil {
			continue
		}
		destination := MNTPath + "/usr/local/bin/" + binary
		if err := copyFile(source, destination, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(source, destination string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		s := filepath.Join(source, entry.Name())
		d := filepath.Join(destination, entry.Name())
		if entry.IsDir() {
			if err := copyTree(s, d); err != nil {
				return err
			}
			continue
		}
		mode := os.FileMode(0o644)
		if info, err := entry.Info(); err == nil && info.Mode().IsRegular() {
			mode = info.Mode().Perm()
		}
		if err := copyFile(s, d, mode); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source, destination string, mode os.FileMode) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, data, mode)
}

// --- system configuration ----------------------------------------------------

// configureSystem mirrors the Python installer: locale, hostname, users,
// services, GRUB and the initramfs.
func configureSystem(cfg *Config, rootPassword, userPassword string) error {
	plog("\n==> Configuring the installed system")

	if err := runStream([]string{"ln", "-sf", "/usr/share/zoneinfo/" + cfg.Timezone,
		MNTPath + "/etc/localtime"}, "", ""); err != nil {
		return fmt.Errorf("setting timezone: %w", err)
	}
	if err := chrootStream([]string{"hwclock", "--systohc"}, "", ""); err != nil {
		warnf("hwclock failed: %v", err)
	}

	if err := writeTargetFile("etc/locale.gen", "en_US.UTF-8 UTF-8\n", 0o644); err != nil {
		return err
	}
	if err := chrootStream([]string{"locale-gen"}, "", ""); err != nil {
		return fmt.Errorf("locale-gen: %w", err)
	}
	if err := writeTargetFile("etc/locale.conf", "LANG=en_US.UTF-8\n", 0o644); err != nil {
		return err
	}

	if err := writeTargetFile("etc/hostname", cfg.Hostname+"\n", 0o644); err != nil {
		return err
	}
	hosts := fmt.Sprintf(
		"127.0.0.1 localhost\n::1 localhost\n127.0.1.1 %s.localdomain %s\n",
		cfg.Hostname, cfg.Hostname)
	if err := writeTargetFile("etc/hosts", hosts, 0o644); err != nil {
		return err
	}

	// The user must exist BEFORE chpasswd is fed (Python parity).
	if err := chrootStream([]string{"useradd", "-m", "-G",
		"wheel,video,audio,storage,network", "-s", "/bin/bash", cfg.Username},
		"", ""); err != nil {
		return fmt.Errorf("useradd: %w", err)
	}

	// Passwords travel over stdin; they never hit the command log.
	credentials := fmt.Sprintf("root:%s\n%s:%s\n", rootPassword, cfg.Username, userPassword)
	if err := chrootStream([]string{"chpasswd"}, credentials, ""); err != nil {
		return fmt.Errorf("chpasswd: %w", err)
	}
	if err := writeTargetFile("etc/sudoers.d/10-wheel", "%wheel ALL=(ALL:ALL) ALL\n", 0o440); err != nil {
		return err
	}

	for _, service := range []string{"NetworkManager.service", "gdm.service"} {
		if err := chrootStream([]string{"systemctl", "enable", service}, "", ""); err != nil {
			warnf("enabling %s failed: %v", service, err)
		}
	}
	if err := chrootStream([]string{"systemctl", "set-default", "graphical.target"}, "", ""); err != nil {
		warnf("setting the default target failed: %v", err)
	}

	if targetExists("/usr/local/bin/kacu-setup") {
		if err := chrootStream([]string{"/usr/local/bin/kacu-setup", "--system"}, "", ""); err != nil {
			warnf("kacu-setup --system failed: %v", err)
		}
	}

	plog("==> Installing GRUB")
	if cfg.UEFI {
		if err := chrootStream([]string{"grub-install", "--target=x86_64-efi",
			"--efi-directory=/boot/efi", "--bootloader-id=kacuOS", "--recheck"}, "", ""); err != nil {
			return fmt.Errorf("grub-install (UEFI): %w", err)
		}
	} else {
		if err := chrootStream([]string{"grub-install", "--target=i386-pc",
			"--recheck", cfg.Disk}, "", ""); err != nil {
			return fmt.Errorf("grub-install (BIOS): %w", err)
		}
	}
	if err := chrootStream([]string{"grub-mkconfig", "-o", "/boot/grub/grub.cfg"}, "", ""); err != nil {
		return fmt.Errorf("grub-mkconfig: %w", err)
	}
	if err := chrootStream([]string{"mkinitcpio", "-P"}, "", ""); err != nil {
		return fmt.Errorf("mkinitcpio: %w", err)
	}
	return nil
}

// chrootStream runs a command inside the new system.
func chrootStream(args []string, stdin, secret string) error {
	return runStream(append([]string{"arch-chroot", MNTPath}, args...), stdin, secret)
}

// chrootCapture captures a chroot command's output.
func chrootCapture(args []string) (string, error) {
	return runCapture(append([]string{"arch-chroot", MNTPath}, args...), "")
}

func targetExists(relative string) bool {
	_, err := os.Stat(filepath.Join(MNTPath, relative))
	return err == nil
}

func writeTargetFile(relative, content string, mode os.FileMode) error {
	full := filepath.Join(MNTPath, relative)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), mode)
}

// --- AUR (Brave) --------------------------------------------------------------

// installAUR builds the AUR packages (e.g. brave-bin) with paru. It installs
// paru itself from a release binary when the target has no helper yet.
func installAUR(cfg *Config) {
	plog("\n==> Installing AUR packages: %s", cfg.BrowserAUR)

	helper := "paru"
	if targetExists("/usr/bin/yay") {
		helper = "yay"
	} else if !targetExists("/usr/bin/paru") && !installParu() {
		warnf("no AUR helper available; skipping %s", cfg.BrowserAUR)
		warnf("after the first boot run: sudo pacman -S yay, then: yay -S %s", cfg.BrowserAUR)
		return
	}

	// Building as root is forbidden; makepkg needs sudo for optional
	// dependency resolution, so grant the user temporary passwordless sudo.
	tempSudoers := "etc/sudoers.d/20-langley-aur"
	if err := writeTargetFile(tempSudoers,
		fmt.Sprintf("%s ALL=(ALL) NOPASSWD: ALL\n", cfg.Username), 0o440); err != nil {
		warnf("could not grant temporary sudo: %v", err)
		return
	}
	command := fmt.Sprintf("%s -S --noconfirm --needed %s", helper, cfg.BrowserAUR)
	if err := chrootStream([]string{
		"su", "-l", cfg.Username, "-c", command}, "", ""); err != nil {
		warnf("AUR build of %s failed: %v", cfg.BrowserAUR, err)
		warnf("install manually after the first boot: %s -S %s", helper, cfg.BrowserAUR)
	}
	_ = os.Remove(filepath.Join(MNTPath, tempSudoers))
}

// installParu downloads the latest released paru binary package and unpacks
// it straight into the target. Returns whether paru then exists there.
func installParu() bool {
	plog("    fetching the paru release binary...")
	index, err := runCapture([]string{"curl", "-fsSL",
		"https://api.github.com/repos/Morganamilo/paru/releases/latest"}, "")
	if err != nil {
		warnf("could not query the paru releases: %v", err)
		return false
	}
	archiveURL := ""
	reURL := regexp.MustCompile(`https://[^"]+\.pkg\.tar\.zst`)
	for _, line := range strings.Split(index, "\n") {
		if !strings.Contains(line, "browser_download_url") {
			continue
		}
		if match := reURL.FindString(line); match != "" && strings.Contains(match, "paru") &&
			strings.Contains(match, "x86_64") {
			archiveURL = match
		}
	}
	if archiveURL == "" {
		warnf("no paru x86_64 .pkg.tar.zst found in the latest release")
		return false
	}
	archive := "/tmp/paru.pkg.tar.zst"
	if err := runStream([]string{"curl", "-fL", "-o", archive, archiveURL}, "", ""); err != nil {
		warnf("downloading paru failed: %v", err)
		return false
	}
	// Try GNU tar with zstd support first, then decompress separately.
	if err := runStream([]string{"tar", "--zstd", "-xf", archive, "-C", MNTPath}, "", ""); err != nil {
		if err := runStream([]string{"zstd", "-q", "-d", "-f", archive,
			"-o", "/tmp/paru.pkg.tar"}, "", ""); err != nil {
			warnf("zstd not available for paru extraction: %v", err)
			return false
		}
		if err := runStream([]string{"tar", "-xf", "/tmp/paru.pkg.tar", "-C", MNTPath}, "", ""); err != nil {
			warnf("unpacking paru failed: %v", err)
			return false
		}
	}
	return targetExists("/usr/bin/paru")
}

// --- boxes ---------------------------------------------------------------------

// installBoxes runs the requested kacu-pkg box presets inside the target.
// Failures are warnings: the presets can be installed after the first boot.
func installBoxes(boxes []string) {
	if len(boxes) == 0 {
		return
	}
	plog("\n==> Installing boxes: %s", strings.Join(boxes, ", "))
	if err := chrootStream(append([]string{"kacu-pkg", "install"}, boxes...), "", ""); err != nil {
		warnf("kacu-pkg failed (%v); run it manually after reboot", err)
	}
}
