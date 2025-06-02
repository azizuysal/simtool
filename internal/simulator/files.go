package simulator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FileInfo represents information about a file or directory
type FileInfo struct {
	Name        string
	Path        string
	Size        int64
	IsDirectory bool
	CreatedAt   time.Time
	ModifiedAt  time.Time
}

// GetFilesForContainer returns all files and directories in the app's data container
func GetFilesForContainer(containerPath string) ([]FileInfo, error) {
	// Remove file:// prefix if present
	if len(containerPath) > 7 && containerPath[:7] == "file://" {
		containerPath = containerPath[7:]
	}
	
	// Check if container path exists
	if _, err := os.Stat(containerPath); err != nil {
		return nil, fmt.Errorf("container path not accessible: %w", err)
	}
	
	entries, err := os.ReadDir(containerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	
	files := make([]FileInfo, 0, len(entries))
	
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't read
		}
		
		fileInfo := FileInfo{
			Name:        entry.Name(),
			Path:        filepath.Join(containerPath, entry.Name()),
			IsDirectory: entry.IsDir(),
			ModifiedAt:  info.ModTime(),
		}
		
		// Get creation time (birth time) - platform specific
		if birthTime := getBirthTime(info); !birthTime.IsZero() {
			fileInfo.CreatedAt = birthTime
		} else {
			// Fallback to modification time if birth time not available
			fileInfo.CreatedAt = info.ModTime()
		}
		
		if !entry.IsDir() {
			fileInfo.Size = info.Size()
		} else {
			// Calculate directory size
			fileInfo.Size = calculateDirSize(fileInfo.Path)
		}
		
		files = append(files, fileInfo)
	}
	
	// Sort: directories first, then alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDirectory != files[j].IsDirectory {
			return files[i].IsDirectory
		}
		return files[i].Name < files[j].Name
	})
	
	return files, nil
}

// FormatFileDate formats a date for display in the file list
func FormatFileDate(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	
	// If modified today, show time
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("Today 15:04")
	}
	
	// If modified this year, show month and day
	if t.Year() == now.Year() {
		return t.Format("Jan 2")
	}
	
	// If within last 7 days, show day of week
	if diff < 7*24*time.Hour && diff > 0 {
		return t.Format("Mon 15:04")
	}
	
	// Otherwise show full date
	return t.Format("Jan 2, 2006")
}