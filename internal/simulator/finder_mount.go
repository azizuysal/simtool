package simulator

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

func isFinderMounted(name string) (bool, error) {
	var child, parent syscall.Stat_t
	if err := syscall.Stat(name, &child); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if err := syscall.Stat(filepath.Dir(name), &parent); err != nil {
		return false, err
	}
	return child.Dev != parent.Dev, nil
}
