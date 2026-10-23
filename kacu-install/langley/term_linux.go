//go:build linux

// term_linux.go — password input without echo via termios (Linux only).
package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// readSecret reads one line with terminal echo disabled.
func readSecret() string {
	fd := int(os.Stdin.Fd())
	var old syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd),
		uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&old))); errno != 0 {
		return readLine()
	}
	noEcho := old
	noEcho.Lflag &^= syscall.ECHO
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd),
		uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&noEcho))); errno != 0 {
		return readLine()
	}
	defer syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd),
		uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&old)))

	line := readLine()
	fmt.Println()
	return line
}
