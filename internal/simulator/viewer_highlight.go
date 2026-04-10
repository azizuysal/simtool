package simulator

import (
	"bytes"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/azizuysal/simtool/internal/config"
)

var (
	// Cache for lexers to improve performance
	lexerCache = make(map[string]chroma.Lexer)
	lexerMutex sync.RWMutex

	// Terminal formatter and style
	termFormatter chroma.Formatter
	chromaStyle   *chroma.Style

	// Initialize once
	initOnce sync.Once
)

// initChromaStyle initializes the chroma style from config
func initChromaStyle() {
	initOnce.Do(func() {
		// Initialize the terminal formatter
		// Try terminal16m first for best color support
		termFormatter = formatters.Get("terminal16m")
		if termFormatter == nil {
			// Fallback to terminal256
			termFormatter = formatters.Get("terminal256")
		}
		if termFormatter == nil {
			// Last resort: basic terminal
			termFormatter = formatters.Get("terminal")
		}

		// Load config
		cfg, err := config.Load()
		if err != nil {
			// Fallback to github-dark if config load fails
			chromaStyle = styles.Get("github-dark")
			return
		}

		// Get the active theme based on config
		themeName := cfg.GetActiveTheme()
		style := styles.Get(themeName)
		if style == nil || style == styles.Fallback {
			// Theme not found, try some variations
			themeLower := strings.ToLower(themeName)
			style = styles.Get(themeLower)

			if style == nil || style == styles.Fallback {
				// Still not found, fallback to github-dark
				style = styles.Get("github-dark")
			}
		}

		chromaStyle = style
	})
}

// GetSyntaxHighlightedLine returns a syntax highlighted version of a line
// This is a simple implementation - could be enhanced with a proper syntax highlighting library
func GetSyntaxHighlightedLine(line string, fileExt string) string {
	return GetSyntaxHighlightedLineWithLang(line, fileExt, "")
}

// GetSyntaxHighlightedLineWithLang returns a syntax highlighted version of a line
// with support for detected language override
func GetSyntaxHighlightedLineWithLang(line string, fileExt string, detectedLang string) string {
	// Initialize style if needed
	initChromaStyle()

	// Quick return for empty lines
	if strings.TrimSpace(line) == "" {
		return line
	}

	// Get or create lexer for this file extension
	var lexer chroma.Lexer

	// If we have a detected language, use that first
	if detectedLang != "" {
		lexer = lexers.Get(detectedLang)
	}

	// Fall back to extension-based detection
	if lexer == nil {
		lexer = getLexerForExtension(fileExt)
	}

	if lexer == nil {
		// No lexer found, return plain text
		return line
	}

	// Tokenize the line
	iterator, err := lexer.Tokenise(nil, line)
	if err != nil {
		return line
	}

	// Format the tokens
	var buf bytes.Buffer
	if termFormatter == nil || chromaStyle == nil {
		// Formatter or style not initialized properly
		return line
	}

	err = termFormatter.Format(&buf, chromaStyle, iterator)
	if err != nil {
		return line
	}

	result := buf.String()
	// If formatting produced no output, return original
	if result == "" {
		return line
	}

	return strings.TrimRight(result, "\n")
}

// detectContentLanguage detects the programming/markup language based on content
func detectContentLanguage(content string) string {
	trimmed := strings.TrimSpace(strings.ToLower(content))

	// Check for HTML patterns
	htmlPatterns := []string{
		"<!doctype html",
		"<html",
		"<head>",
		"<body>",
		"<div",
		"<span",
		"<p>",
		"<h1",
		"<h2",
		"<h3",
		"<meta",
		"<title>",
		"<script",
		"<style",
		"<link",
	}

	for _, pattern := range htmlPatterns {
		if strings.Contains(trimmed, pattern) {
			return "html"
		}
	}

	// Check for SVG patterns first
	if strings.HasPrefix(trimmed, "<?xml") {
		// If it starts with XML declaration, check if it's SVG
		if strings.Contains(trimmed, "<svg") || strings.Contains(trimmed, "xmlns=\"http://www.w3.org/2000/svg\"") {
			return "svg"
		}
	}

	// Check for XML patterns (but not HTML or SVG)
	if strings.HasPrefix(trimmed, "<?xml") ||
		(strings.Contains(trimmed, "<") && strings.Contains(trimmed, ">") &&
			!strings.Contains(trimmed, "<html") && !strings.Contains(trimmed, "<body") &&
			!strings.Contains(trimmed, "<svg")) {
		// Simple XML detection - has tags but not HTML or SVG tags
		return "xml"
	}

	// Check for JSON patterns
	if (strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, ":")) ||
		(strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "{")) {
		return "json"
	}

	return ""
}

// getLexerForExtension returns a cached lexer for the given file extension
func getLexerForExtension(fileExt string) chroma.Lexer {
	// Try to get from cache first
	lexerMutex.RLock()
	lexer, exists := lexerCache[fileExt]
	lexerMutex.RUnlock()

	if exists {
		return lexer
	}

	// Create new lexer
	lexerMutex.Lock()
	defer lexerMutex.Unlock()

	// Check again in case another goroutine created it
	if lexer, exists := lexerCache[fileExt]; exists {
		return lexer
	}

	// Get lexer by filename (chroma uses the extension)
	lexer = lexers.Match("file" + fileExt)
	if lexer == nil {
		// Try some common aliases
		switch fileExt {
		case ".h":
			lexer = lexers.Get("c")
		case ".hpp", ".hxx":
			lexer = lexers.Get("cpp")
		case ".m":
			lexer = lexers.Get("objective-c")
		case ".mm":
			lexer = lexers.Get("objective-c++")
			if lexer == nil {
				// Fallback to Objective-C if Objective-C++ not available
				lexer = lexers.Get("objective-c")
			}
			if lexer == nil {
				// Final fallback to C++
				lexer = lexers.Get("cpp")
			}
		case ".yml":
			lexer = lexers.Get("yaml")
		case ".tsx":
			lexer = lexers.Get("typescript")
		case ".jsx":
			lexer = lexers.Get("react")
		case ".plist":
			lexer = lexers.Get("xml")
		case ".htm", ".html":
			lexer = lexers.Get("html")
		case ".podspec":
			lexer = lexers.Get("ruby")
		}
	}

	if lexer != nil {
		// Clone the lexer to avoid concurrent modification issues
		lexer = chroma.Coalesce(lexer)
		lexerCache[fileExt] = lexer
	}

	return lexer
}
