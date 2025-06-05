package config

import (
	"fmt"
	"os"
	"strings"
	"time"
	
	"golang.org/x/term"
)

// QueryTerminalBackgroundColor queries the terminal's background color using OSC 11
func QueryTerminalBackgroundColor() (string, error) {
	// Check if stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("not running in a terminal")
	}
	
	// Save current terminal state
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("getting terminal state: %w", err)
	}
	
	// Ensure we restore terminal on exit
	defer func() {
		term.Restore(int(os.Stdin.Fd()), oldState)
	}()
	
	// Enter raw mode
	_, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("entering raw mode: %w", err)
	}
	
	// Send OSC 11 query with ST terminator (ESC\)
	fmt.Fprintf(os.Stdout, "\033]11;?\033\\")
	os.Stdout.Sync()
	
	// Read response with timeout
	responseChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		buf := make([]byte, 256)
		n, err := os.Stdin.Read(buf)
		if err != nil {
			errorChan <- err
		} else {
			responseChan <- buf[:n]
		}
	}()
	
	select {
	case response := <-responseChan:
		// Restore terminal before processing
		term.Restore(int(os.Stdin.Fd()), oldState)
		return parseOSC11Response(string(response)), nil
	case err := <-errorChan:
		return "", err
	case <-time.After(200 * time.Millisecond):
		// Many terminals don't support OSC 11
		return "", fmt.Errorf("timeout waiting for response")
	}
}

// parseOSC11Response parses the OSC 11 response
// Format: ESC]11;rgb:RRRR/GGGG/BBBB ESC\ or BEL
func parseOSC11Response(response string) string {
	// Look for rgb: pattern
	start := strings.Index(response, "rgb:")
	if start == -1 {
		return ""
	}
	
	// Find the end (ESC\ or BEL)
	end := len(response)
	for i := start; i < len(response); i++ {
		if response[i] == '\033' || response[i] == '\007' {
			end = i
			break
		}
	}
	
	return response[start:end]
}

// IsColorDark determines if an RGB color string is dark
func IsColorDark(colorStr string) bool {
	// Parse "rgb:RRRR/GGGG/BBBB" format
	if !strings.HasPrefix(colorStr, "rgb:") {
		return true // Default to dark if we can't parse
	}
	
	parts := strings.Split(strings.TrimPrefix(colorStr, "rgb:"), "/")
	if len(parts) != 3 {
		return true
	}
	
	// Parse hex values (often 16-bit per channel)
	var r, g, b uint64
	fmt.Sscanf(parts[0], "%x", &r)
	fmt.Sscanf(parts[1], "%x", &g)
	fmt.Sscanf(parts[2], "%x", &b)
	
	// Normalize to 0-255 range
	// Terminal colors can be 8-bit (FF) or 16-bit (FFFF)
	if r > 255 { r = r >> 8 }
	if g > 255 { g = g >> 8 }
	if b > 255 { b = b >> 8 }
	
	// Calculate perceived brightness using ITU-R BT.709 formula
	brightness := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255.0
	
	// Consider dark if brightness is less than 0.5
	return brightness < 0.5
}