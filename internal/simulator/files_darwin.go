//go:build darwin
// +build darwin

package simulator

import (
	"os"
	"syscall"
	"time"
)

// getBirthTime gets the creation time (birth time) on macOS
func getBirthTime(info os.FileInfo) time.Time {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}
	}
	
	// On macOS, Birthtimespec contains the creation time
	return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
}