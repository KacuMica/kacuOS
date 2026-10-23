// ui.go — logging, subprocess helpers and terminal prompts (stdlib only).
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	MNTPath = "/mnt"
	LogPath = "/var/log/langley.log"
)

var (
	stdin     = bufio.NewReader(os.Stdin)
	logHandle *os.File
)

// --- logging ---------------------------------------------------------------

// OpenLog truncates and opens the log file; logging continues without it on error.
func OpenLog() {
	f, err := os.OpenFile(LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err == nil {
		logHandle = f
	}
}

// plog prints to stdout and mirrors the line into the log file.
func plog(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
	if logHandle != nil {
		fmt.Fprintf(logHandle, format+"\n", args...)
	}
}

// warnf logs a WARNING line to stdout and the log file.
func warnf(format string, args ...any) {
	plog("WARNING: "+format, args...)
}

// dief logs an ERROR line and exits with status 1.
func dief(format string, args ...any) {
	plog("ERROR: "+format, args...)
	if logHandle != nil {
		_ = logHandle.Sync()
	}
	os.Exit(1)
}

// --- subprocess helpers ----------------------------------------------------

// ExecOpt tunes a single runCmd call.
type ExecOpt struct {
	Stdin   string // data piped into the command (never logged)
	Secret  string // replaced with *** in the log line
	Capture bool   // capture stdout+stderr instead of streaming
}

// runCmd executes cmd. When Capture is set the combined output is returned,
// otherwise the command writes directly to our stdout/stderr.
func runCmd(cmd []string, opt ExecOpt) (string, error) {
	cmdLine := strings.Join(cmd, " ")
	if opt.Secret != "" {
		cmdLine = maskSecret(cmdLine, opt.Secret)
	}
	plog("+ %s", cmdLine)

	c := exec.Command(cmd[0], cmd[1:]...)
	if opt.Stdin != "" {
		c.Stdin = strings.NewReader(opt.Stdin)
	}
	var buf bytes.Buffer
	if opt.Capture {
		c.Stdout = &buf
		c.Stderr = &buf
	} else {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}
	err := c.Run()
	if err != nil {
		out := strings.TrimSpace(buf.String())
		if opt.Capture && out != "" {
			plog("%s", out)
		}
		return out, err
	}
	return strings.TrimSpace(buf.String()), nil
}

// runStream feeds stdin, streams output to the console and returns the error.
func runStream(cmd []string, stdin, secret string) error {
	_, err := runCmd(cmd, ExecOpt{Stdin: stdin, Secret: secret})
	return err
}

// runCapture captures stdout+stderr; the bool is true on success.
func runCapture(cmd []string, secret string) (string, error) {
	return runCmd(cmd, ExecOpt{Capture: true, Secret: secret})
}

// maskSecret hides sensitive arguments (e.g. Wi-Fi passphrases) in the log.
func maskSecret(line, secret string) string {
	if secret == "" {
		return line
	}
	return strings.ReplaceAll(line, secret, "***")
}

// commandExists reports whether a program is on PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// requireRoot mirrors the Python installer's first check.
func requireRoot() {
	if os.Geteuid() != 0 {
		dief("langley must run as root (try: sudo langley)")
	}
}

// --- input -----------------------------------------------------------------

// readLine reads one trimmed line from stdin. At end of input it exits
// cleanly — otherwise single-choice menus would spin forever on EOF.
func readLine() string {
	line, err := stdin.ReadString('\n')
	if line == "" && err != nil {
		dief("end of input (stdin closed)")
	}
	return strings.TrimSpace(line)
}

// prompt asks until a non-empty answer is given; def is accepted via Enter.
func prompt(label, def string) string {
	for {
		if def != "" {
			fmt.Printf("%s [%s]: ", label, def)
		} else {
			fmt.Printf("%s: ", label)
		}
		answer := readLine()
		if answer != "" {
			return answer
		}
		if def != "" {
			return def
		}
		fmt.Println("please enter a value")
	}
}

// promptPassword reads a password twice without echo and requires a match.
func promptPassword(label string) string {
	for {
		fmt.Printf("%s: ", label)
		first := readSecret()
		fmt.Printf("Repeat %s: ", strings.ToLower(label))
		second := readSecret()
		if first == "" {
			fmt.Println("password cannot be empty")
			continue
		}
		if first != second {
			fmt.Println("passwords do not match")
			continue
		}
		return first
	}
}

// menu prints a numbered single-choice menu and returns the 1-based pick.
// With allowZero, pressing Enter (or 0) returns 0 = back/cancel.
func menu(title string, options []string, allowZero bool) int {
	plog("\n%s", title)
	for i, option := range options {
		plog("  %d) %s", i+1, option)
	}
	for {
		suffix := "choice"
		if allowZero {
			suffix = "choice (0 = back)"
		}
		fmt.Printf("%s: ", suffix)
		line := readLine()
		if line == "" && allowZero {
			return 0
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(options) {
			fmt.Println("invalid choice")
			continue
		}
		return n
	}
}

// multiMenu prints a checkbox list. Type numbers (space-separated) to toggle,
// an empty line finishes. It returns the 0-based indexes left enabled.
func multiMenu(title string, items []string, defaults map[int]bool) []int {
	enabled := map[int]bool{}
	for i, on := range defaults {
		enabled[i] = on
	}
	for {
		fmt.Printf("\n%s\n", title)
		for i, item := range items {
			mark := " "
			if enabled[i] {
				mark = "x"
			}
			fmt.Printf("  [%s] %d) %s\n", mark, i+1, item)
		}
		fmt.Print("numbers to toggle (space-separated), empty line to continue: ")
		line := readLine()
		if line == "" {
			break
		}
		for _, field := range strings.Fields(line) {
			n, err := strconv.Atoi(field)
			if err != nil || n < 1 || n > len(items) {
				fmt.Printf("invalid number: %s\n", field)
				continue
			}
			enabled[n-1] = !enabled[n-1]
		}
	}
	var chosen []int
	for i := range items {
		if enabled[i] {
			chosen = append(chosen, i)
		}
	}
	return chosen
}

// confirmYes asks a yes/no question.
func confirmYes(question string) bool {
	fmt.Printf("%s [y/N]: ", question)
	answer := strings.ToLower(readLine())
	return answer == "y" || answer == "yes"
}
