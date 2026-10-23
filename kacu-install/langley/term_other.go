//go:build !linux

// term_other.go — password input fallback for non-Linux builds (build check).
package main

// readSecret falls back to a plain visible line outside Linux.
func readSecret() string {
	return readLine()
}
