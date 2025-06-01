package simulator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// App represents an installed application
type App struct {
	Name      string
	BundleID  string
	Version   string
	Size      int64
	Path      string
	Container string
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