package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MemInfo struct {
	MemTotalKB     int64
	MemAvailableKB int64
	SwapTotalKB    int64
	SwapFreeKB     int64
}

type CpuInfo struct {
	ModelName string
	CpuCount  int
}

type LoadInfo struct {
	Load1  float64
	Load5  float64
	Load15 float64
}

func meminfoReadField(path, key string) (int64, bool) {
	file, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, key) {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, err := strconv.ParseInt(parts[1], 10, 64)
				if err == nil {
					return val, true
				}
			}
		}
	}
	return 0, false
}

func readMemInfo() (MemInfo, bool) {
	var mem MemInfo
	var ok1, ok2 bool

	mem.MemTotalKB, ok1 = meminfoReadField("/proc/meminfo", "MemTotal:")
	mem.MemAvailableKB, ok2 = meminfoReadField("/proc/meminfo", "MemAvailable:")
	mem.SwapTotalKB, _ = meminfoReadField("/proc/meminfo", "SwapTotal:")
	mem.SwapFreeKB, _ = meminfoReadField("/proc/meminfo", "SwapFree:")

	return mem, ok1 && ok2
}

func readCpuInfo() CpuInfo {
	cpu := CpuInfo{ModelName: "unknown", CpuCount: 0}

	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return cpu
	}
	defer file.Close()

	gotModel := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !gotModel && strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cpu.ModelName = strings.TrimSpace(parts[1])
				gotModel = true
			}
		}
		if strings.HasPrefix(line, "processor") {
			cpu.CpuCount++
		}
	}
	return cpu
}

func readLoadInfo() LoadInfo {
	load := LoadInfo{}
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return load
	}
	defer file.Close()

	var l1, l5, l15 float64
	_, err = fmt.Fscanf(file, "%f %f %f", &l1, &l5, &l15)
	if err == nil {
		load.Load1 = l1
		load.Load5 = l5
		load.Load15 = l15
	}
	return load
}

func printPlain(mem MemInfo, cpu CpuInfo, load LoadInfo) {
	fmt.Println("kacuOS system info")
	fmt.Println("------------------")
	fmt.Printf("CPU model:    %s\n", cpu.ModelName)
	fmt.Printf("CPU cores:    %d\n", cpu.CpuCount)
	fmt.Printf("Memory total: %.1f MB\n", float64(mem.MemTotalKB)/1024.0)
	fmt.Printf("Memory used:  %.1f MB\n", float64(mem.MemTotalKB-mem.MemAvailableKB)/1024.0)
	fmt.Printf("Memory avail: %.1f MB\n", float64(mem.MemAvailableKB)/1024.0)

	if mem.SwapTotalKB > 0 {
		fmt.Printf("Swap total:   %.1f MB\n", float64(mem.SwapTotalKB)/1024.0)
		fmt.Printf("Swap free:    %.1f MB\n", float64(mem.SwapFreeKB)/1024.0)
	} else {
		fmt.Println("Swap:         none (or zram not yet active)")
	}
	fmt.Printf("Load avg:     %.2f %.2f %.2f\n", load.Load1, load.Load5, load.Load15)
}

func printJSON(mem MemInfo, cpu CpuInfo, load LoadInfo) {
	fmt.Println("{")
	fmt.Printf("  \"cpu_model\": \"%s\",\n", cpu.ModelName)
	fmt.Printf("  \"cpu_cores\": %d,\n", cpu.CpuCount)
	fmt.Printf("  \"mem_total_kb\": %d,\n", mem.MemTotalKB)
	fmt.Printf("  \"mem_available_kb\": %d,\n", mem.MemAvailableKB)
	fmt.Printf("  \"mem_used_kb\": %d,\n", mem.MemTotalKB-mem.MemAvailableKB)
	fmt.Printf("  \"swap_total_kb\": %d,\n", mem->SwapTotalKB)
	fmt.Printf("  \"swap_free_kb\": %d,\n", mem->SwapFreeKB)
	fmt.Printf("  \"load1\": %.2f,\n", load.Load1)
	fmt.Printf("  \"load5\": %.2f,\n", load.Load5)
	fmt.Printf("  \"load15\": %.2f\n", load.Load15)
	fmt.Println("}")
}

func main() {
	jsonMode := false
	if len(os.Args) > 1 && os.Args[1] == "--json" {
		jsonMode = true
	}

	mem, ok := readMemInfo()
	if !ok {
		fmt.Fprintln(os.Stderr, "kacu-sysinfo: failed to read /proc/meminfo")
		os.Exit(1)
	}

	cpu := readCpuInfo()
	load := readLoadInfo()

	if jsonMode {
		printJSON(mem, cpu, load)
	} else {
		printPlain(mem, cpu, load)
	}
}
