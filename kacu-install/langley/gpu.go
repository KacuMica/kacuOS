// gpu.go — Step 4: GPU detection and driver package selection.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// gpuDriver is one selectable driver set.
type gpuDriver struct {
	id    string
	label string
	pkgs  []string
}

var gpuDrivers = []gpuDriver{
	{"intel", "Intel graphics (Mesa + VA-API/Vulkan)",
		[]string{"mesa", "vulkan-intel", "intel-media-driver", "libva-intel-driver"}},
	{"amd", "AMD/ATI graphics (Mesa + Vulkan/VDPAU)",
		[]string{"mesa", "vulkan-radeon", "libva-mesa-driver", "mesa-vdpau"}},
	{"nvidia", "NVIDIA proprietary (matches the stock linux kernel)",
		[]string{"nvidia", "nvidia-utils", "nvidia-settings"}},
	{"nvidia-dkms", "NVIDIA proprietary, DKMS (builds for any kernel)",
		[]string{"nvidia-dkms", "nvidia-utils", "nvidia-settings", "linux-headers"}},
	{"none", "No extra GPU drivers (Mesa only)", nil},
}

// gpuVendor is a detected graphics controller (lowercase vendor id family).
type gpuVendor struct {
	id   string
	desc string
}

// gpuStep shows detections and lets the user toggle driver sets (multi-select
// so laptops with hybrid graphics can pick Intel + NVIDIA).
func gpuStep(cfg *Config) {
	var detected []gpuVendor
	if out, err := runCapture([]string{"lspci", "-nn"}, ""); err == nil {
		detected = parseLspciGPUs(out)
	} else {
		detected = gpuVendorsSysfs()
	}
	for _, v := range detected {
		plog("Detected GPU: %s", v.desc)
	}
	if len(detected) == 0 {
		plog("GPU detection found nothing; plain Mesa will be used")
	}

	defaults := map[int]bool{}
	if len(detected) == 0 {
		if idx := gpuDriverIndex("none"); idx >= 0 {
			defaults[idx] = true
		}
	}
	for _, v := range detected {
		// gpuDriverIndex("nvidia") picks the plain set, not the DKMS one.
		if idx := gpuDriverIndex(v.id); idx >= 0 {
			defaults[idx] = true
		}
	}

	chosen := multiMenu("Graphics drivers", gpuDriverLabels(), defaults)
	var pkgs []string
	var ids []string
	for _, idx := range chosen {
		driver := gpuDrivers[idx]
		if driver.id == "none" {
			continue
		}
		ids = append(ids, driver.id)
		pkgs = append(pkgs, driver.pkgs...)
	}
	cfg.GPUIDs = ids
	cfg.GPU = dedupe(pkgs)
}

// parseLspciGPUs extracts VGA/3D/display controllers with their vendor ids.
func parseLspciGPUs(out string) []gpuVendor {
	var found []gpuVendor
	seen := map[string]bool{}
	reID := regexp.MustCompile(`\[([0-9a-fA-F]{4}):([0-9a-fA-F]{4})\]`)
	for _, line := range strings.Split(out, "\n") {
		low := strings.ToLower(line)
		isGPU := strings.Contains(low, "vga") ||
			strings.Contains(low, "3d controller") ||
			strings.Contains(low, "display controller")
		if !isGPU {
			continue
		}
		id := ""
		if m := reID.FindStringSubmatch(line); m != nil {
			id = vendorFromID(strings.ToLower(m[1]))
		} else {
			switch {
			case strings.Contains(low, "nvidia"):
				id = "nvidia"
			case strings.Contains(low, "intel"):
				id = "intel"
			case strings.Contains(low, "amd"), strings.Contains(low, "ati"):
				id = "amd"
			}
		}
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		found = append(found, gpuVendor{id: id, desc: strings.TrimSpace(line)})
	}
	return found
}

// gpuVendorsSysfs falls back to /sys/class/drm when lspci is unavailable.
func gpuVendorsSysfs() []gpuVendor {
	entries, err := os.ReadDir("/sys/class/drm")
	if err != nil {
		return nil
	}
	var found []gpuVendor
	seen := map[string]bool{}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join("/sys/class/drm", entry.Name(),
			"device", "vendor"))
		if err != nil {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(data)), "0x"))
		id := vendorFromID(strings.ToLower(raw))
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		found = append(found, gpuVendor{id: id, desc: fmt.Sprintf("GPU vendor 0x%s (%s)", raw, id)})
	}
	return found
}

// vendorFromID maps a PCI vendor id to the driver family id.
func vendorFromID(hexVendor string) string {
	switch hexVendor {
	case "8086":
		return "intel"
	case "1002":
		return "amd"
	case "10de":
		return "nvidia"
	}
	return ""
}

func gpuDriverLabels() []string {
	labels := make([]string, len(gpuDrivers))
	for i, driver := range gpuDrivers {
		if len(driver.pkgs) > 0 {
			labels[i] = fmt.Sprintf("%s — %s", driver.label, strings.Join(driver.pkgs, ", "))
		} else {
			labels[i] = driver.label
		}
	}
	return labels
}

func gpuDriverIndex(id string) int {
	for i := range gpuDrivers {
		if gpuDrivers[i].id == id {
			return i
		}
	}
	return -1
}

// cpuVendor reads vendor_id from /proc/cpuinfo (Intel/AMD) for microcode.
func cpuVendor() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "vendor_id") {
			fields := strings.Split(line, ":")
			if len(fields) >= 2 {
				return strings.TrimSpace(fields[1])
			}
		}
	}
	return ""
}
