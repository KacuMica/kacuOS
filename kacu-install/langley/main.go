// ==========================================================================
// langley — interactive Arch Linux installer for kacuOS (Go rewrite)
//
// Runs inside the kacuOS live ISO (needs root). It is the archinstall-style
// companion to the Python `kacu_install`: it adds a network setup wizard,
// GPU driver selection, an audio stack choice and a software/box picker on
// top of the classic disk -> pacstrap -> configure -> GRUB flow.
//
// Usage:
//
//	sudo langley                        # fully interactive
//	sudo langley --disk /dev/sda        # preselect the target disk
//	sudo langley --list-disks
//	sudo langley --disk /dev/sda --yes --dry-run
//
// WARNING: the selected disk is completely erased.
// ==========================================================================
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const langVersion = "1.0.0"

// availableBoxes mirrors the kacu-pkg presets (dev/server/ai).
var availableBoxes = []string{"dev", "server", "ai"}

// Config holds every choice the user makes during the run.
type Config struct {
	Disk     string
	UEFI     bool
	Hostname string
	Username string
	Timezone string

	GPUIDs      []string // chosen driver ids (intel/amd/nvidia/nvidia-dkms)
	GPU         []string // driver packages appended to the base list
	AudioID     string   // pipewire / pulse / alsa
	BrowserRepo string   // firefox or chromium ("" = none)
	BrowserAUR  string   // brave-bin ("" = none)

	Steam bool
	Apps  []string

	Boxes     []string
	AURNeeded bool // true when BrowserAUR is set (needs base-devel in pacstrap)
}

func main() {
	var (
		preDisk       = flag.String("disk", "", "target disk, e.g. /dev/sda or /dev/nvme0n1")
		hostnameFlag  = flag.String("hostname", "", "hostname for the installed system")
		usernameFlag  = flag.String("username", "", "user account to create (wheel + sudo)")
		timezoneFlag  = flag.String("timezone", "", "IANA timezone, e.g. Europe/Kyiv")
		boxesFlag     = flag.String("boxes", "", "comma-separated boxes: dev,server,ai")
		forceUEFI     = flag.Bool("uefi", false, "force a UEFI install")
		forceBIOS     = flag.Bool("bios", false, "force a BIOS install")
		yesFlag       = flag.Bool("yes", false, "skip the final YES confirmation")
		dryRun        = flag.Bool("dry-run", false, "print the plan and exit")
		listDisksFlag = flag.Bool("list-disks", false, "list disks and exit")
		versionFlag   = flag.Bool("version", false, "print the version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "langley — interactive kacuOS installer (Arch Linux base)")
		fmt.Fprintln(os.Stderr, "Erases the target disk. Logs: /var/log/langley.log")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionFlag {
		fmt.Printf("langley %s\n", langVersion)
		return
	}

	requireRoot()

	plog("langley %s — kacuOS installer", langVersion)
	plog("=================================================")
	OpenLog()

	if *listDisksFlag {
		for _, line := range listDisks() {
			fmt.Println(line)
		}
		return
	}

	if _, err := os.Stat(MNTPath); err != nil || !commandExists("pacstrap") {
		warnf("this does not look like the kacuOS live environment; continuing anyway")
	}

	// --- Step 1: network ---------------------------------------------------
	plog("\nStep 1/8 — Network")
	networkStep()

	cfg := &Config{}

	// --- Step 2: disk + firmware -------------------------------------------
	plog("\nStep 2/8 — Disk and firmware")
	cfg.Disk = chooseDisk(*preDisk)
	if *forceUEFI && *forceBIOS {
		dief("--uefi and --bios are mutually exclusive")
	}
	cfg.UEFI = *forceUEFI || (!*forceBIOS && detectUEFI())

	// --- Step 3: identity ---------------------------------------------------
	plog("\nStep 3/8 — System identity")
	cfg.Hostname = *hostnameFlag
	if cfg.Hostname == "" {
		cfg.Hostname = prompt("Hostname", "kacuos")
	}
	cfg.Username = strings.ToLower(strings.TrimSpace(*usernameFlag))
	if cfg.Username == "" {
		cfg.Username = strings.ToLower(prompt("Username", "kacu"))
	}
	cfg.Timezone = *timezoneFlag
	if cfg.Timezone == "" {
		cfg.Timezone = prompt("Timezone (e.g. Europe/Kyiv)", "UTC")
	}
	validateTimezone(cfg.Timezone)
	cfg.Boxes = parseBoxesFlag(*boxesFlag)

	// --- Step 4: graphics drivers ------------------------------------------
	plog("\nStep 4/8 — Graphics drivers")
	gpuStep(cfg)

	// --- Step 5: audio -------------------------------------------------------
	plog("\nStep 5/8 — Audio")
	audioStep(cfg)

	// --- Step 6: software ----------------------------------------------------
	plog("\nStep 6/8 — Software")
	softwareStep(cfg)
	cfg.AURNeeded = cfg.BrowserAUR != ""

	// --- Step 7: plan + confirm ----------------------------------------------
	pkgList := buildPackages(cfg)
	plog("\nStep 7/8 — Plan")
	printPlan(cfg, len(pkgList))
	if *dryRun {
		plog("\nDry run: stopping before any destructive change.")
		return
	}
	if !*yesFlag {
		fmt.Print("Type YES to continue: ")
		if readLine() != "YES" {
			fmt.Println("Cancelled.")
			os.Exit(1)
		}
	}
	rootPW := promptPassword("Root password")
	userPW := promptPassword(fmt.Sprintf("Password for %s", cfg.Username))

	// --- Step 8: install ------------------------------------------------------
	plog("\nStep 8/8 — Installing")
	runInstall(cfg, rootPW, userPW)
}

// parseBoxesFlag validates the --boxes flag against availableBoxes.
func parseBoxesFlag(value string) []string {
	var boxes []string
	if strings.TrimSpace(value) == "" {
		return boxes
	}
	for _, b := range strings.Split(value, ",") {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		if !validBox(b) {
			dief("unknown box: %s (available: %s)", b, strings.Join(availableBoxes, ","))
		}
		boxes = append(boxes, b)
	}
	return boxes
}

func validBox(box string) bool {
	for _, candidate := range availableBoxes {
		if candidate == box {
			return true
		}
	}
	return false
}

func validateTimezone(tz string) {
	if _, err := os.Stat("/usr/share/zoneinfo/" + tz); err != nil {
		dief("unknown timezone: %s", tz)
	}
}

// printPlan shows every decision before anything destructive happens.
func printPlan(cfg *Config, pkgCount int) {
	plog("\nPlan")
	plog("----")
	plog("  disk:      %s  (%s)", cfg.Disk, diskInfo(cfg.Disk))
	plog("  firmware:  %s", firmwareLabel(cfg.UEFI))
	plog("  hostname:  %s", cfg.Hostname)
	plog("  user:      %s (wheel, sudo with password)", cfg.Username)
	plog("  timezone:  %s", cfg.Timezone)
	plog("  audio:     %s", audioLabel(cfg.AudioID))
	plog("  gpu:       %s", joinOrNone(cfg.GPU))
	plog("  browser:   %s", browserLabel(cfg))
	plog("  steam:     %t", cfg.Steam)
	plog("  apps:      %s", joinOrNone(cfg.Apps))
	plog("  boxes:     %s", joinOrNone(cfg.Boxes))
	plog("  packages:  %d", pkgCount)
	plog("")
	plog("ALL DATA ON THE SELECTED DISK WILL BE DESTROYED.")
}

func firmwareLabel(uefi bool) string {
	if uefi {
		return "UEFI"
	}
	return "BIOS"
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}

func browserLabel(cfg *Config) string {
	parts := make([]string, 0, 2)
	if cfg.BrowserRepo != "" {
		parts = append(parts, cfg.BrowserRepo)
	}
	if cfg.BrowserAUR != "" {
		parts = append(parts, cfg.BrowserAUR+" (AUR)")
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, " + ")
}
