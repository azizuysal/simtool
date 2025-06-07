package simulator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// App represents an installed application
type App struct {
	Name           string
	BundleID       string
	Version        string
	Size           int64
	Path           string
	Container      string
	SimulatorName  string    // Name of the parent simulator
	SimulatorUDID  string    // UDID of the parent simulator
	ModTime        time.Time // Last modified time of the app
}

// GetAppsForSimulator returns all apps installed on a simulator
func GetAppsForSimulator(udid string, isRunning bool) ([]App, error) {
	if isRunning {
		return getAppsFromListApps(udid)
	}
	return getAppsFromDataDir(udid)
}

// getAppsFromListApps gets apps for running simulators
func getAppsFromListApps(udid string) ([]App, error) {
	cmd := exec.Command("xcrun", "simctl", "listapps", udid)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}

	// Parse the plist-style output
	apps := make([]App, 0)
	lines := strings.Split(string(output), "\n")
	
	var currentApp App
	inApp := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Start of a new app entry
		if strings.HasPrefix(line, `"`) && strings.Contains(line, " = ") && strings.HasSuffix(line, "{") {
			// Extract bundle ID from lines like: "com.example.app" =     {
			parts := strings.SplitN(line, " = ", 2)
			if len(parts) == 2 {
				bundleID := strings.Trim(parts[0], `"`)
				if !strings.HasPrefix(bundleID, "com.apple.") {
					currentApp = App{BundleID: bundleID}
					inApp = true
				} else {
					inApp = false
				}
			}
		} else if inApp {
			if strings.HasPrefix(line, "CFBundleDisplayName = ") {
				currentApp.Name = strings.Trim(strings.TrimPrefix(line, "CFBundleDisplayName = "), `";`)
			} else if strings.HasPrefix(line, "CFBundleShortVersionString = ") {
				currentApp.Version = strings.Trim(strings.TrimPrefix(line, "CFBundleShortVersionString = "), `";`)
			} else if strings.HasPrefix(line, "Path = ") {
				// Path values are not quoted in the output
				currentApp.Path = strings.TrimSpace(strings.TrimPrefix(line, "Path = "))
				currentApp.Path = strings.TrimSuffix(currentApp.Path, ";")
			} else if strings.HasPrefix(line, "DataContainer = ") {
				currentApp.Container = strings.Trim(strings.TrimPrefix(line, "DataContainer = "), `";`)
			} else if line == "};" && currentApp.BundleID != "" {
				// Calculate app size from path
				if currentApp.Path != "" {
					currentApp.Size = calculateDirSize(currentApp.Path)
					// Get modification time
					if info, err := os.Stat(currentApp.Path); err == nil {
						currentApp.ModTime = info.ModTime()
					}
				}
				// Use bundle ID as name if display name is empty
				if currentApp.Name == "" {
					currentApp.Name = currentApp.BundleID
				}
				apps = append(apps, currentApp)
				inApp = false
			}
		}
	}
	
	// Sort apps by name
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})
	
	return apps, nil
}

// getAppsFromDataDir gets apps for non-running simulators
func getAppsFromDataDir(udid string) ([]App, error) {
	homeDir := os.Getenv("HOME")
	appPath := fmt.Sprintf("%s/Library/Developer/CoreSimulator/Devices/%s/data/Containers/Bundle/Application", homeDir, udid)
	
	entries, err := os.ReadDir(appPath)
	if err != nil {
		// Directory might not exist if no apps installed
		if os.IsNotExist(err) {
			return []App{}, nil
		}
		return nil, fmt.Errorf("failed to read app directory: %w", err)
	}
	
	apps := make([]App, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			appDir := filepath.Join(appPath, entry.Name())
			// Look for .app directory inside
			appEntries, err := os.ReadDir(appDir)
			if err != nil {
				continue
			}
			
			for _, appEntry := range appEntries {
				if strings.HasSuffix(appEntry.Name(), ".app") {
					app := App{
						Path: filepath.Join(appDir, appEntry.Name()),
					}
					
					// Try to read app info from Info.plist
					if info := readAppInfo(app.Path); info != nil {
						app.Name = info.DisplayName
						app.BundleID = info.BundleID
						app.Version = info.Version
						if app.Name == "" {
							app.Name = strings.TrimSuffix(appEntry.Name(), ".app")
						}
					} else {
						app.Name = strings.TrimSuffix(appEntry.Name(), ".app")
						app.BundleID = "Unknown"
					}
					
					app.Size = calculateDirSize(app.Path)
					
					// Get modification time
					if info, err := os.Stat(app.Path); err == nil {
						app.ModTime = info.ModTime()
					}
					
					// For non-running simulators, we need to find the data container
					// It's in a different location based on the bundle ID
					dataPath := fmt.Sprintf("%s/Library/Developer/CoreSimulator/Devices/%s/data/Containers/Data/Application", homeDir, udid)
					if app.BundleID != "" && app.BundleID != "Unknown" {
						// Try to find the data container for this app
						app.Container = findDataContainer(dataPath, app.BundleID)
					}
					
					apps = append(apps, app)
					break
				}
			}
		}
	}
	
	// Sort apps by name
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})
	
	return apps, nil
}

// AppInfo represents parsed Info.plist data
type AppInfo struct {
	DisplayName string
	BundleID    string
	Version     string
}

// readAppInfo reads basic info from Info.plist
func readAppInfo(appPath string) *AppInfo {
	plistPath := filepath.Join(appPath, "Info.plist")
	cmd := exec.Command("plutil", "-convert", "json", "-o", "-", plistPath)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	
	var plist map[string]interface{}
	if err := json.Unmarshal(output, &plist); err != nil {
		return nil
	}
	
	info := &AppInfo{}
	if v, ok := plist["CFBundleDisplayName"].(string); ok {
		info.DisplayName = v
	} else if v, ok := plist["CFBundleName"].(string); ok {
		info.DisplayName = v
	}
	
	if v, ok := plist["CFBundleIdentifier"].(string); ok {
		info.BundleID = v
	}
	
	if v, ok := plist["CFBundleShortVersionString"].(string); ok {
		info.Version = v
	}
	
	return info
}

// calculateDirSize calculates the size of a directory
func calculateDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// findDataContainer finds the data container for an app by its bundle ID
func findDataContainer(dataPath string, bundleID string) string {
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return ""
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			containerPath := filepath.Join(dataPath, entry.Name())
			// Check if .com.apple.mobile_container_manager.metadata.plist exists
			metadataPath := filepath.Join(containerPath, ".com.apple.mobile_container_manager.metadata.plist")
			
			// Try to read the metadata to verify this is the right container
			cmd := exec.Command("plutil", "-convert", "json", "-o", "-", metadataPath)
			output, err := cmd.Output()
			if err != nil {
				continue
			}
			
			var metadata map[string]interface{}
			if err := json.Unmarshal(output, &metadata); err != nil {
				continue
			}
			
			// Check if this container belongs to our app
			if identifier, ok := metadata["MCMMetadataIdentifier"].(string); ok && identifier == bundleID {
				return containerPath
			}
		}
	}
	
	return ""
}

// FormatSize formats bytes into human readable format
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatModTime formats modification time in a human-friendly way
func FormatModTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	
	now := time.Now()
	diff := now.Sub(t)
	
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 48*time.Hour:
		return "yesterday"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	default:
		// For older dates, show the actual date
		if t.Year() == now.Year() {
			return t.Format("Jan 2")
		}
		return t.Format("Jan 2, 2006")
	}
}

// appInfo represents the JSON structure from xcrun simctl listapps
type appInfo struct {
	ApplicationType            string `json:"ApplicationType"`
	Bundle                     string `json:"Bundle"`
	BundleContainer            string `json:"BundleContainer"`
	CFBundleDisplayName        string `json:"CFBundleDisplayName"`
	CFBundleIdentifier         string `json:"CFBundleIdentifier"`
	CFBundleName               string `json:"CFBundleName"`
	CFBundleShortVersionString string `json:"CFBundleShortVersionString"`
	DataContainer              string `json:"DataContainer"`
	Path                       string `json:"Path"`
}

// parseAppListJSON parses the JSON output from xcrun simctl listapps
func parseAppListJSON(data []byte) ([]App, error) {
	var appMap map[string]appInfo
	if err := json.Unmarshal(data, &appMap); err != nil {
		return nil, fmt.Errorf("failed to parse app list JSON: %w", err)
	}

	apps := make([]App, 0)
	for bundleID, info := range appMap {
		// Skip system apps
		if info.ApplicationType != "User" {
			continue
		}

		app := App{
			BundleID:  bundleID,
			Version:   info.CFBundleShortVersionString,
			Path:      info.Path,
			Container: info.DataContainer,
			Size:      0, // Size is calculated separately
		}

		// Determine app name
		if info.CFBundleDisplayName != "" {
			app.Name = info.CFBundleDisplayName
		} else if info.CFBundleName != "" {
			app.Name = info.CFBundleName
		} else {
			app.Name = bundleID
		}

		apps = append(apps, app)
	}

	return apps, nil
}