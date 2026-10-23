// disk.go — Step 2: disk discovery, partitioning, formatting and mounting.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// listDisks returns lsblk lines for whole disks only.
func listDisks() []string {
	out, err := runCapture([]string{"lsblk", "-dpno", "NAME,SIZE,MODEL,TYPE"}, "")
	if err != nil {
		dief("lsblk failed: %v", err)
	}
	var disks []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 || fields[len(fields)-1] != "disk" {
			continue
		}
		name := filepath.Base(fields[0])
		// zram devices report type "disk"; they are no install targets.
		if strings.HasPrefix(name, "zram") || strings.HasPrefix(name, "loop") ||
			strings.HasPrefix(name, "ram") {
			continue
		}
		disks = append(disks, line)
	}
	return disks
}

// diskInfo returns a human summary of one disk (SIZE, MODEL).
func diskInfo(disk string) string {
	out, err := runCapture([]string{"lsblk", "-dno", "SIZE,MODEL", disk}, "")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// chooseDisk validates a preselected disk or shows the interactive menu,
// refusing partitions and disks with mounted partitions.
func chooseDisk(preselected string) string {
	disks := listDisks()
	if len(disks) == 0 {
		dief("no disks found")
	}
	plog("\nAvailable disks:")
	for i, line := range disks {
		plog("  %d) %s", i+1, line)
	}

	disk := preselected
	if disk == "" {
		fmt.Print("\nSelect the target disk number: ")
		answer := readLine()
		n, err := strconv.Atoi(answer)
		if err != nil || n < 1 || n > len(disks) {
			dief("invalid selection: %s", answer)
		}
		disk = strings.Fields(disks[n-1])[0]
	}

	if _, err := os.Stat(disk); os.IsNotExist(err) {
		dief("%s does not exist", disk)
	}

	kind, err := runCapture([]string{"lsblk", "-dno", "TYPE", disk}, "")
	trimmed := strings.TrimSpace(kind)
	if err != nil || (trimmed != "disk" && trimmed != "loop") {
		dief("%s is not a whole disk", disk)
	}

	if mounted, err := runCapture([]string{"lsblk", "-no", "MOUNTPOINT", disk}, ""); err == nil &&
		strings.TrimSpace(mounted) != "" {
		dief("%s has mounted partitions; unmount them first", disk)
	}
	return disk
}

// partName adds the 'p' separator for disks whose names end in a digit
// (/dev/nvme0n1, /dev/mmcblk0).
func partName(disk string, number int) string {
	last := disk[len(disk)-1]
	if last >= '0' && last <= '9' {
		return fmt.Sprintf("%sp%d", disk, number)
	}
	return fmt.Sprintf("%s%d", disk, number)
}

// detectUEFI checks for the efivars filesystem.
func detectUEFI() bool {
	info, err := os.Stat("/sys/firmware/efi")
	return err == nil && info.IsDir()
}

// partitionDisk wipes the disk and creates the GPT layout (this erases it).
func partitionDisk(disk string, uefi bool) error {
	plog("\n==> Partitioning (this erases the disk)")
	if err := runStream([]string{"sgdisk", "--zap-all", disk}, "", ""); err != nil {
		return fmt.Errorf("sgdisk --zap-all: %w", err)
	}
	if uefi {
		if err := runStream([]string{"sgdisk", "-n", "1:0:+512M", "-t", "1:ef00",
			"-c", "1:kacuOS ESP", disk}, "", ""); err != nil {
			return fmt.Errorf("sgdisk ESP: %w", err)
		}
	} else {
		if err := runStream([]string{"sgdisk", "-n", "1:0:+1M", "-t", "1:ef02",
			"-c", "1:BIOS boot", disk}, "", ""); err != nil {
			return fmt.Errorf("sgdisk BIOS boot: %w", err)
		}
	}
	if err := runStream([]string{"sgdisk", "-n", "2:0:0", "-t", "2:8300",
		"-c", "2:kacuOS root", disk}, "", ""); err != nil {
		return fmt.Errorf("sgdisk root: %w", err)
	}
	_, _ = runCapture([]string{"partprobe", disk}, "")
	time.Sleep(1 * time.Second)
	return nil
}

// formatAndMount formats the ESP + root partitions and mounts them under /mnt.
func formatAndMount(disk string, uefi bool) error {
	plog("\n==> Formatting and mounting")
	rootPart := partName(disk, 2)
	if uefi {
		espPart := partName(disk, 1)
		if err := runStream([]string{"mkfs.fat", "-F32", "-n", "KACU_ESP", espPart}, "", ""); err != nil {
			return fmt.Errorf("mkfs.fat: %w", err)
		}
	}
	if err := runStream([]string{"mkfs.ext4", "-F", "-L", "kacuOS", rootPart}, "", ""); err != nil {
		return fmt.Errorf("mkfs.ext4: %w", err)
	}

	if err := os.MkdirAll(MNTPath, 0o755); err != nil {
		return err
	}
	if err := runStream([]string{"mount", rootPart, MNTPath}, "", ""); err != nil {
		return fmt.Errorf("mount root: %w", err)
	}
	if uefi {
		espMount := MNTPath + "/boot/efi"
		if err := os.MkdirAll(espMount, 0o755); err != nil {
			return err
		}
		if err := runStream([]string{"mount", partName(disk, 1), espMount}, "", ""); err != nil {
			return fmt.Errorf("mount ESP: %w", err)
		}
	}
	return nil
}

// unmountAll is deferred by runInstall; three tries with a delay between them.
func unmountAll() {
	plog("\n==> Unmounting")
	for attempt := 0; attempt < 3; attempt++ {
		if err := runStream([]string{"umount", "-R", MNTPath}, "", ""); err == nil {
			return
		}
		time.Sleep(1 * time.Second)
	}
}
