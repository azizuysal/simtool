//go:build !darwin
// +build !darwin

package simulator

import (
	"os"
	"time"
)

// getBirthTime returns zero time on non-macOS platforms
func getBirthTime(info os.FileInfo) time.Time {
	return time.Time{}
}