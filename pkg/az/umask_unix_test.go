//go:build !windows

package az

import "golang.org/x/sys/unix"

// setUmask sets the process umask and returns the previous value.
func setUmask(mask int) int { return unix.Umask(mask) }
