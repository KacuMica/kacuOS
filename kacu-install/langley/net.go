// net.go — Step 1: internet connection setup via NetworkManager (nmcli).
package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// wifiNet is one scanned wireless network.
type wifiNet struct {
	inUse    bool
	ssid     string
	signal   int
	security string // "" or "--" means open
}

// networkStep blocks until the machine is online; pacstrap needs the net.
func networkStep() {
	if checkOnline() {
		plog("Network: online")
		return
	}
	if !commandExists("nmcli") {
		dief("no internet and NetworkManager is unavailable; " +
			"connect manually (e.g. iwctl) and re-run langley")
	}
	for {
		plog("Network: offline")
		switch menu("Network setup", []string{
			"Connect Ethernet (DHCP via NetworkManager)",
			"Connect to a Wi-Fi network",
			"Re-check connectivity",
			"Abort the installer",
		}, false) {
		case 1:
			connectEthernet()
		case 2:
			connectWiFi()
		case 3:
			// fall through to the re-check below
		case 4:
			dief("aborted by user (network is required to download packages)")
		}
		if checkOnline() {
			plog("Network: online")
			return
		}
	}
}

// checkOnline tries ping and falls back to curl against archlinux.org.
func checkOnline() bool {
	for _, host := range []string{"1.1.1.1", "9.9.9.9"} {
		if _, err := runCapture([]string{"ping", "-c", "1", "-W", "4", host}, ""); err == nil {
			return true
		}
	}
	if commandExists("curl") {
		if _, err := runCapture([]string{"curl", "-fsSI", "--max-time", "8",
			"https://archlinux.org"}, ""); err == nil {
			return true
		}
	}
	return false
}

// --- Ethernet ---------------------------------------------------------------

// connectEthernet picks a wired device and brings it up with DHCP.
func connectEthernet() {
	out, err := runCapture([]string{"nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "device"}, "")
	if err != nil {
		warnf("nmcli failed: %v", err)
		return
	}
	var candidates []string
	for _, line := range strings.Split(out, "\n") {
		fields := splitUnescapedColon(line)
		if len(fields) >= 2 && fields[1] == "ethernet" {
			candidates = append(candidates, fields[0])
		}
	}
	if len(candidates) == 0 {
		warnf("no ethernet devices found")
		return
	}
	dev := candidates[0]
	if len(candidates) > 1 {
		choice := menu("Ethernet devices", candidates, true)
		if choice == 0 {
			return
		}
		dev = candidates[choice-1]
	}
	plog("Bringing up %s (DHCP)...", dev)
	if err := runStream([]string{"nmcli", "device", "connect", dev}, "", ""); err != nil {
		warnf("could not bring up %s: %v", dev, err)
	}
}

// --- Wi-Fi ------------------------------------------------------------------

// connectWiFi scans and connects to a wireless network of the user's choice.
func connectWiFi() {
	iface := pickWifiIface()
	if iface == "" {
		return
	}
	for {
		_, _ = runCapture([]string{"nmcli", "device", "wifi", "rescan", "ifname", iface}, "")
		time.Sleep(2 * time.Second)

		out, err := runCapture([]string{"nmcli", "-t", "-f",
			"IN-USE,SSID,SIGNAL,SECURITY", "device", "wifi", "list", "ifname", iface}, "")
		if err != nil {
			warnf("wifi scan failed: %v", err)
			return
		}
		nets := parseWiFi(out)
		if len(nets) == 0 {
			warnf("no networks found; rescanning")
			time.Sleep(1 * time.Second)
			continue
		}

		options := make([]string, len(nets))
		for i, net := range nets {
			mark := " "
			if net.inUse {
				mark = ">"
			}
			sec := net.security
			if sec == "" {
				sec = "open"
			}
			options[i] = fmt.Sprintf("%s %-34s %3d%%  %s", mark, net.ssid, net.signal, sec)
		}
		choice := menu(fmt.Sprintf("Wi-Fi networks (%s)", iface), options, true)
		if choice == 0 {
			return
		}
		sel := nets[choice-1]

		pass := ""
		if sel.security != "" && sel.security != "--" {
			fmt.Printf("Password for %s: ", sel.ssid)
			pass = readSecret()
		}
		cmd := []string{"nmcli", "device", "wifi", "connect", sel.ssid, "ifname", iface}
		if pass != "" {
			cmd = append(cmd, "password", pass)
		}
		if err := runStream(cmd, "", pass); err == nil {
			return
		} else {
			warnf("could not connect to %s: %v", sel.ssid, err)
		}
		if !confirmYes("Retry?") {
			return
		}
	}
}

// pickWifiIface returns a wireless interface name (interactive when several).
func pickWifiIface() string {
	out, err := runCapture([]string{"nmcli", "-t", "-f", "DEVICE,TYPE", "device"}, "")
	if err != nil {
		warnf("nmcli failed: %v", err)
		return ""
	}
	var ifaces []string
	for _, line := range strings.Split(out, "\n") {
		fields := splitUnescapedColon(line)
		if len(fields) >= 2 && fields[1] == "wifi" {
			ifaces = append(ifaces, fields[0])
		}
	}
	if len(ifaces) == 0 {
		warnf("no wifi interfaces found")
		return ""
	}
	if len(ifaces) == 1 {
		return ifaces[0]
	}
	choice := menu("Wifi interfaces", ifaces, true)
	if choice == 0 {
		return ""
	}
	return ifaces[choice-1]
}

// parseWiFi reads `nmcli -t -f IN-USE,SSID,SIGNAL,SECURITY device wifi list`.
func parseWiFi(out string) []wifiNet {
	var nets []wifiNet
	for _, line := range strings.Split(out, "\n") {
		fields := splitUnescapedColon(line)
		if len(fields) < 4 {
			continue
		}
		ssid := strings.TrimSpace(fields[1])
		if ssid == "" {
			continue // hidden network: connect explicitly instead
		}
		signal, _ := strconv.Atoi(strings.TrimSpace(fields[2]))
		nets = append(nets, wifiNet{
			inUse:    strings.HasPrefix(strings.TrimSpace(fields[0]), "*"),
			ssid:     ssid,
			signal:   signal,
			security: strings.TrimSpace(fields[3]),
		})
	}
	return nets
}

// splitUnescapedColon splits a terse nmcli line on unescaped ':' characters.
// nmcli escapes colons inside SSIDs as '\:'; those are preserved (unescaped).
func splitUnescapedColon(s string) []string {
	var fields []string
	var current strings.Builder
	escaped := false
	for _, r := range s {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == ':':
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	return append(fields, current.String())
}
