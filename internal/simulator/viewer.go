package simulator

import (
	"archive/zip"
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	_ "github.com/mattn/go-sqlite3"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	_ "golang.org/x/image/webp" // Add WebP support
	
	"simtool/internal/config"
)

// FileType represents the type of file for viewing
type FileType int

const (
	FileTypeText FileType = iota
	FileTypeImage
	FileTypeBinary
	FileTypeArchive
	FileTypeDatabase
)

// FileContent represents the content of a file prepared for viewing
type FileContent struct {
	Type        FileType
	Lines       []string // For text files
	TotalLines  int      // Total number of lines in the file
	ImageInfo   *ImageInfo
	BinaryData  []byte   // For hex view (current chunk)
	BinaryOffset int64   // Offset of the current chunk in the file
	TotalSize   int64    // Total size of the file (for binary files)
	ArchiveInfo *ArchiveInfo // For archive files
	DatabaseInfo *DatabaseInfo // For database files
	IsBinaryPlist bool    // Whether this was converted from binary plist
	DetectedLang string  // Detected language for syntax highlighting (e.g., "html")
	Error       error
}

// ImageInfo contains metadata about an image file
type ImageInfo struct {
	Format  string
	Width   int
	Height  int
	Size    int64
	Preview *ImagePreview
}

// ImagePreview contains the terminal-renderable preview
type ImagePreview struct {
	Width  int
	Height int
	Rows   []string // Pre-rendered rows with ANSI colors
}

// ArchiveInfo contains information about an archive file
type ArchiveInfo struct {
	Format         string
	Entries        []ArchiveEntry
	FileCount      int
	FolderCount    int
	TotalSize      int64
	CompressedSize int64
}

// ArchiveEntry represents a single file or directory in an archive
type ArchiveEntry struct {
	Name         string
	Size         int64
	CompressedSize int64
	ModTime      time.Time
	IsDir        bool
}

// DatabaseInfo contains information about a database file
type DatabaseInfo struct {
	Format      string      `json:"format"`       // "SQLite", "MySQL", etc.
	Version     string      `json:"version"`      // Database version
	FileSize    int64       `json:"file_size"`
	TableCount  int         `json:"table_count"`
	Tables      []TableInfo `json:"tables"`
	Schema      string      `json:"schema"`       // Full schema dump
	Error       string      `json:"error,omitempty"`
}

// TableInfo represents information about a database table
type TableInfo struct {
	Name     string            `json:"name"`
	RowCount int64             `json:"row_count"`
	Schema   string            `json:"schema"`
	Columns  []ColumnInfo      `json:"columns"`
	Sample   []map[string]any  `json:"sample,omitempty"` // First few rows
}

// ColumnInfo represents information about a table column
type ColumnInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	NotNull bool   `json:"not_null"`
	PK      bool   `json:"primary_key"`
}

// DetectFileType determines the type of file based on content and extension
func DetectFileType(path string) FileType {
	// First check by extension
	ext := strings.ToLower(filepath.Ext(path))
	
	// Image file extensions
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".webp": true, ".ico": true, ".svg": true,
	}
	
	if imageExts[ext] {
		return FileTypeImage
	}
	
	// Archive file extensions
	archiveExts := map[string]bool{
		".zip": true, ".jar": true, ".war": true, ".ear": true,
		".ipa": true, ".apk": true, ".aar": true,
	}
	
	if archiveExts[ext] {
		return FileTypeArchive
	}
	
	// Database file extensions
	databaseExts := map[string]bool{
		".db": true, ".sqlite": true, ".sqlite3": true, ".db3": true,
	}
	
	if databaseExts[ext] {
		return FileTypeDatabase
	}
	
	// Check for known binary extensions first
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".dat": true, ".cache": true,
		".o": true, ".a": true, ".lib": true, ".obj": true,
		".class": true, ".jar": true, ".dex": true,
		".pyc": true, ".pyo": true, ".wasm": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true,
		".xlsx": true, ".ppt": true, ".pptx": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
		".wav": true, ".flac": true, ".ogg": true, ".m4a": true,
		".ttf": true, ".otf": true, ".woff": true, ".woff2": true,
		".eot": true, ".pfb": true, ".pfm": true,
	}
	
	if binaryExts[ext] {
		return FileTypeBinary
	}
	
	// For non-binary extensions, check content
	file, err := os.Open(path)
	if err != nil {
		return FileTypeBinary
	}
	defer file.Close()
	
	// Read first 512 bytes to check content
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return FileTypeBinary
	}
	
	// Check for SQLite magic header "SQLite format 3\000"
	if n >= 16 && string(buffer[:15]) == "SQLite format 3" {
		return FileTypeDatabase
	}
	
	// Check for image file signatures
	imageSignatures := [][]byte{
		[]byte("\x89PNG\r\n\x1a\n"),              // PNG
		[]byte("\xFF\xD8\xFF"),                    // JPEG
		[]byte("GIF87a"), []byte("GIF89a"),       // GIF
		[]byte("RIFF"),                            // RIFF (might be WebP)
		[]byte("BM"),                              // BMP
		[]byte("MM\x00\x2A"), []byte("II\x2A\x00"), // TIFF
		[]byte("\x00\x00\x01\x00"),                // ICO
		[]byte("\x00\x00\x02\x00"),                // CUR
	}
	
	for _, sig := range imageSignatures {
		if n >= len(sig) && bytes.Equal(buffer[:len(sig)], sig) {
			// For RIFF, check if it's WebP
			if bytes.Equal(sig, []byte("RIFF")) && n >= 12 {
				if bytes.Equal(buffer[8:12], []byte("WEBP")) {
					return FileTypeImage
				}
				// Otherwise it might be WAV or AVI, treat as binary
				return FileTypeBinary
			}
			return FileTypeImage
		}
	}
	
	// Check for archive signatures
	archiveSignatures := [][]byte{
		[]byte("PK\x03\x04"), []byte("PK\x05\x06"), // ZIP and variants
		[]byte("\x1F\x8B"),                         // GZIP
		[]byte("BZh"),                              // BZIP2
		[]byte("\xFD7zXZ\x00"),                     // XZ
		[]byte("Rar!"),                             // RAR
	}
	
	for _, sig := range archiveSignatures {
		if n >= len(sig) && bytes.Equal(buffer[:len(sig)], sig) {
			return FileTypeArchive
		}
	}
	
	// Check if the content is valid UTF-8 and mostly printable
	if isTextContent(buffer[:n]) {
		return FileTypeText
	}
	
	// For unknown extensions with non-text content, still check common text extensions
	// This helps with files that might have encoding issues in the first 512 bytes
	textExts := map[string]bool{
		".txt": true, ".md": true, ".log": true, ".json": true,
		".xml": true, ".yaml": true, ".yml": true, ".toml": true,
		".go": true, ".js": true, ".ts": true, ".py": true,
		".java": true, ".c": true, ".cpp": true, ".h": true,
		".swift": true, ".m": true, ".mm": true, ".rb": true,
		".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".css": true, ".html": true, ".htm": true, ".vue": true,
		".jsx": true, ".tsx": true, ".rs": true, ".plist": true,
		".gitignore": true, ".env": true, ".conf": true, ".ini": true,
		".podspec": true, ".gemspec": true, ".rake": true, ".gemfile": true,
		".podfile": true, ".brewfile": true, ".rakefile": true,
	}
	
	if textExts[ext] {
		return FileTypeText
	}
	
	// Files without extensions should be treated as binary unless content check passed
	return FileTypeBinary
}

// isTextContent checks if the content appears to be text
func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	
	// Check for known binary file signatures/magic bytes
	binarySignatures := [][]byte{
		[]byte("bplist"),                          // Binary plist
		[]byte("\x89PNG\r\n\x1a\n"),              // PNG
		[]byte("\xFF\xD8\xFF"),                    // JPEG
		[]byte("GIF87a"), []byte("GIF89a"),       // GIF
		[]byte("RIFF"),                            // RIFF (WebP, WAV, AVI)
		[]byte("\x00\x00\x01\x00"),                // ICO
		[]byte("\x00\x00\x02\x00"),                // CUR
		[]byte("BM"),                              // BMP
		[]byte("MM\x00\x2A"), []byte("II\x2A\x00"), // TIFF
		[]byte("PK\x03\x04"), []byte("PK\x05\x06"), // ZIP and variants
		[]byte("\xCA\xFE\xBA\xBE"),                // Mach-O binary
		[]byte("\xCE\xFA\xED\xFE"),                // Mach-O binary (32-bit)
		[]byte("\xCF\xFA\xED\xFE"),                // Mach-O binary (64-bit)
		[]byte("\xFE\xED\xFA\xCE"),                // Mach-O binary (big-endian)
		[]byte("\xFE\xED\xFA\xCF"),                // Mach-O binary (64-bit big-endian)
		[]byte("SQLite format 3"),                 // SQLite database
		[]byte("\x1F\x8B"),                        // GZIP
		[]byte("BZh"),                             // BZIP2
		[]byte("\xFD7zXZ\x00"),                    // XZ
		[]byte("Rar!"),                            // RAR
		[]byte("\x50\x4B"),                        // PKZip
		[]byte("\x7FELF"),                         // ELF binary
		[]byte("%PDF-"),                           // PDF
		[]byte("\x25\x50\x44\x46\x2D"),            // PDF (hex)
		[]byte("\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1"), // Microsoft Office (doc, xls, ppt)
		[]byte("\x50\x4B\x03\x04"),                // Microsoft Office (docx, xlsx, pptx)
		[]byte("\x4F\x67\x67\x53"),                // OGG
		[]byte("\x38\x42\x50\x53"),                // PSD
		[]byte("\x52\x49\x46\x46"),                // WAV
		[]byte("\x00\x00\x00\x0C\x6A\x50\x20\x20"), // JPEG 2000
		[]byte("\x1A\x45\xDF\xA3"),                // MKV, WebM
		[]byte("\x00\x00\x00\x14\x66\x74\x79\x70"), // MP4, M4V, M4A
		[]byte("\x49\x44\x33"),                    // MP3
	}
	
	// Check against known signatures
	for _, sig := range binarySignatures {
		if len(data) >= len(sig) && bytes.Equal(data[:len(sig)], sig) {
			return false
		}
	}
	
	// Check if it's valid UTF-8
	if !utf8.Valid(data) {
		return false
	}
	
	// Count printable vs non-printable characters
	printable := 0
	nullBytes := 0
	for _, b := range data {
		// Count null bytes as a strong indicator of binary content
		if b == 0 {
			nullBytes++
		}
		// Allow printable ASCII, newlines, tabs, carriage returns
		if (b >= 32 && b <= 126) || b == '\n' || b == '\t' || b == '\r' {
			printable++
		}
	}
	
	// If there are null bytes, likely binary
	if nullBytes > 0 {
		return false
	}
	
	// If more than 90% of characters are printable, consider it text
	return float64(printable)/float64(len(data)) > 0.9
}

// ReadFileContent reads file content based on its type
func ReadFileContent(path string, startLine, maxLines, maxWidth int) (*FileContent, error) {
	fileType := DetectFileType(path)
	
	content := &FileContent{
		Type: fileType,
	}
	
	switch fileType {
	case FileTypeText:
		lines, totalLines, isBinaryPlist, err := readTextFile(path, startLine, maxLines)
		content.Lines = lines
		content.TotalLines = totalLines
		content.IsBinaryPlist = isBinaryPlist
		content.Error = err
		
		// Detect language for files without extensions
		if filepath.Ext(path) == "" && len(lines) > 0 {
			// Check first few lines for content type
			checkContent := strings.Join(lines, "\n")
			if len(checkContent) > 500 {
				checkContent = checkContent[:500]
			}
			if lang := detectContentLanguage(checkContent); lang != "" {
				content.DetectedLang = lang
				// If we detected SVG content, switch to image handling
				if lang == "svg" {
					content.Type = FileTypeImage
					info, err := readImageInfo(path, maxLines, maxWidth)
					if err != nil && strings.Contains(err.Error(), "not a valid image") {
						// If SVG parsing fails, keep as text with SVG syntax highlighting
						content.Type = FileTypeText
						content.DetectedLang = "xml" // Use XML highlighting for SVG
					} else {
						content.ImageInfo = info
						content.Error = err
						return content, content.Error
					}
				}
			}
		}
		
	case FileTypeImage:
		info, err := readImageInfo(path, maxLines, maxWidth) // Pass dimensions for preview size
		if err != nil && strings.Contains(err.Error(), "not a valid image") {
			// Fall back to binary view if image decoding fails
			content.Type = FileTypeBinary
			
			// Get file info
			fileInfo, err := os.Stat(path)
			if err != nil {
				content.Error = err
				return content, err
			}
			
			content.TotalSize = fileInfo.Size()
			
			// Calculate the offset based on startLine (each line = 16 bytes)
			offset := int64(startLine * 16)
			
			// Read a chunk of data (8KB) starting from the offset
			const chunkSize = 8192 // 8KB chunks
			readSize := chunkSize
			
			// Don't read past the end of the file
			if offset+int64(readSize) > fileInfo.Size() {
				readSize = int(fileInfo.Size() - offset)
			}
			
			if readSize > 0 {
				data, err := readBinaryFile(path, offset, readSize)
				content.BinaryData = data
				content.BinaryOffset = offset
				content.Error = err
			} else {
				content.BinaryData = []byte{}
				content.BinaryOffset = offset
			}
			
			return content, content.Error
		}
		content.ImageInfo = info
		content.Error = err
		
	case FileTypeBinary:
		// For binary files, implement lazy loading
		fileInfo, err := os.Stat(path)
		if err != nil {
			content.Error = err
			return content, err
		}
		
		content.TotalSize = fileInfo.Size()
		
		// Calculate the offset based on startLine (each line = 16 bytes)
		offset := int64(startLine * 16)
		
		// Read a chunk of data (8KB) starting from the offset
		const chunkSize = 8192 // 8KB chunks
		readSize := chunkSize
		
		// Don't read past the end of the file
		if offset+int64(readSize) > fileInfo.Size() {
			readSize = int(fileInfo.Size() - offset)
		}
		
		if readSize > 0 {
			data, err := readBinaryFile(path, offset, readSize)
			content.BinaryData = data
			content.BinaryOffset = offset
			content.Error = err
		} else {
			content.BinaryData = []byte{}
			content.BinaryOffset = offset
		}
		
	case FileTypeArchive:
		info, err := readArchiveInfo(path)
		content.ArchiveInfo = info
		content.Error = err
		
	case FileTypeDatabase:
		info, err := readDatabaseInfo(path)
		content.DatabaseInfo = info
		content.Error = err
	}
	
	return content, content.Error
}

// readTextFile reads a text file with pagination support
// Returns lines, totalLines, isBinaryPlist, error
func readTextFile(path string, startLine, maxLines int) ([]string, int, bool, error) {
	isBinaryPlist := false
	
	// Check if it's a plist file
	if strings.HasSuffix(strings.ToLower(path), ".plist") {
		// Check if it's a binary plist by reading first few bytes
		file, err := os.Open(path)
		if err != nil {
			return nil, 0, false, err
		}
		
		magic := make([]byte, 6)
		n, err := file.Read(magic)
		file.Close()
		
		if err == nil && n >= 6 && string(magic) == "bplist" {
			// It's a binary plist, convert it to XML for viewing
			lines, total, err := readBinaryPlist(path, startLine, maxLines)
			return lines, total, true, err
		}
		// Otherwise, fall through to read as normal text file
	}
	
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, false, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	var lines []string
	currentLine := 0
	totalLines := 0
	
	// Set a larger buffer size for very long lines (common in minified files)
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)
	
	for scanner.Scan() {
		totalLines++
		if currentLine >= startLine && len(lines) < maxLines {
			line := scanner.Text()
			// Truncate very long lines for display
			if len(line) > 2000 {
				line = line[:2000] + "..."
			}
			lines = append(lines, line)
		}
		currentLine++
	}
	
	if err := scanner.Err(); err != nil {
		return lines, totalLines, isBinaryPlist, err
	}
	
	return lines, totalLines, isBinaryPlist, nil
}

// readBinaryPlist converts a binary plist to XML and reads it
func readBinaryPlist(path string, startLine, maxLines int) ([]string, int, error) {
	// Use plutil to convert binary plist to XML
	cmd := exec.Command("plutil", "-convert", "xml1", "-o", "-", path)
	output, err := cmd.Output()
	if err != nil {
		// If conversion fails, return an error message
		return []string{fmt.Sprintf("Error converting binary plist: %v", err)}, 1, nil
	}
	
	// Split the XML output into lines
	allLines := strings.Split(string(output), "\n")
	totalLines := len(allLines)
	
	// Handle empty output
	if totalLines == 0 {
		return []string{}, 0, nil
	}
	
	// Apply pagination
	endLine := startLine + maxLines
	if endLine > totalLines {
		endLine = totalLines
	}
	
	lines := make([]string, 0, endLine-startLine)
	for i := startLine; i < endLine; i++ {
		line := allLines[i]
		// Truncate very long lines for display
		if len(line) > 2000 {
			line = line[:2000] + "..."
		}
		lines = append(lines, line)
	}
	
	return lines, totalLines, nil
}

// readImageInfo reads image metadata and generates preview
func readImageInfo(path string, maxPreviewHeight, maxPreviewWidth int) (*ImageInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	
	// Check if it's an SVG file by extension or content
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".svg" {
		return readSVGInfo(path, stat.Size(), maxPreviewHeight, maxPreviewWidth)
	}
	
	// For files without extension, check if content looks like SVG
	if ext == "" {
		// Read first 512 bytes to check for SVG content
		buffer := make([]byte, 512)
		n, err := file.Read(buffer)
		if err == nil && n > 0 {
			content := strings.ToLower(string(buffer[:n]))
			if strings.Contains(content, "<?xml") && 
			   (strings.Contains(content, "<svg") || strings.Contains(content, "xmlns=\"http://www.w3.org/2000/svg\"")) {
				// Reset file position
				file.Close()
				return readSVGInfo(path, stat.Size(), maxPreviewHeight, maxPreviewWidth)
			}
		}
		// Reset file position for image decoding
		file.Seek(0, 0)
	}
	
	// Decode image to get dimensions
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("not a valid image: %w", err)
	}
	
	info := &ImageInfo{
		Format: format,
		Width:  config.Width,
		Height: config.Height,
		Size:   stat.Size(),
	}
	
	// Generate preview if requested
	if maxPreviewHeight > 15 { // Only generate preview if we have reasonable space
		// Reset file position
		file.Seek(0, 0)
		
		// Decode full image for preview
		img, _, err := image.Decode(file)
		if err == nil {
			// Calculate available space
			// The maxPreviewHeight already accounts for UI overhead from model.go
			// Just reserve 4 lines for the image info header in the viewer
			availableHeight := maxPreviewHeight - 4
			if availableHeight > 0 {
				// Use the actual available width minus some padding
				// Account for content box padding (4 chars)
				availableWidth := maxPreviewWidth - 4
				if availableWidth < 20 {
					availableWidth = 20
				}
				info.Preview = generateImagePreview(img, availableWidth, availableHeight)
			}
		}
	}
	
	return info, nil
}

// readBinaryFile reads a chunk of binary file
func readBinaryFile(path string, offset int64, size int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	// Seek to offset
	_, err = file.Seek(offset, 0)
	if err != nil {
		return nil, err
	}
	
	// Read chunk
	data := make([]byte, size)
	n, err := file.Read(data)
	if err != nil && err != io.EOF {
		return nil, err
	}
	
	// Return only the bytes actually read, not the full buffer
	return data[:n], nil
}

// FormatHexDump formats binary data as hex dump
func FormatHexDump(data []byte, offset int64) []string {
	var lines []string
	
	for i := 0; i < len(data); i += 16 {
		// Address
		line := fmt.Sprintf("%08x  ", offset+int64(i))
		
		// Hex bytes
		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				line += fmt.Sprintf("%02x ", data[i+j])
			} else {
				line += "   "
			}
			// Extra space in the middle
			if j == 7 {
				line += " "
			}
		}
		
		line += " |"
		
		// ASCII representation
		for j := 0; j < 16 && i+j < len(data); j++ {
			b := data[i+j]
			if b >= 32 && b <= 126 {
				line += string(b)
			} else {
				line += "."
			}
		}
		line += "|"
		
		lines = append(lines, line)
	}
	
	return lines
}

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

// readArchiveInfo reads information about an archive file
func readArchiveInfo(path string) (*ArchiveInfo, error) {
	// Open the ZIP file
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %v", err)
	}
	defer reader.Close()
	
	info := &ArchiveInfo{
		Format:  "ZIP",
		Entries: make([]ArchiveEntry, 0, len(reader.File)),
	}
	
	// Read all entries and calculate statistics
	for _, file := range reader.File {
		entry := ArchiveEntry{
			Name:           file.Name,
			Size:           int64(file.UncompressedSize64),
			CompressedSize: int64(file.CompressedSize64),
			ModTime:        file.Modified,
			IsDir:          file.FileInfo().IsDir(),
		}
		info.Entries = append(info.Entries, entry)
		
		if entry.IsDir {
			info.FolderCount++
		} else {
			info.FileCount++
			info.TotalSize += entry.Size
			info.CompressedSize += entry.CompressedSize
		}
	}
	
	return info, nil
}

// generateImagePreview creates a terminal-renderable preview of an image
func generateImagePreview(img image.Image, maxWidth, maxHeight int) *ImagePreview {
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()
	
	// Calculate target dimensions maintaining aspect ratio
	// We use half-blocks, so each character can show 2 vertical pixels
	aspectRatio := float64(imgWidth) / float64(imgHeight)
	
	// Target dimensions in characters (height will use half-blocks for 2x resolution)
	var targetWidth, targetHeight int
	
	// Account for half-blocks: actual pixel height is 2x character height
	effectiveMaxHeight := maxHeight * 2
	
	if aspectRatio > float64(maxWidth)/float64(effectiveMaxHeight) {
		// Width-constrained
		targetWidth = maxWidth
		targetHeight = int(float64(maxWidth) / aspectRatio)
	} else {
		// Height-constrained
		targetHeight = effectiveMaxHeight
		targetWidth = int(float64(effectiveMaxHeight) * aspectRatio)
	}
	
	// Ensure even height for half-blocks
	if targetHeight%2 != 0 {
		targetHeight--
	}
	
	// Character dimensions
	charHeight := targetHeight / 2
	charWidth := targetWidth
	
	// Create preview
	preview := &ImagePreview{
		Width:  charWidth,
		Height: charHeight,
		Rows:   make([]string, charHeight),
	}
	
	// Scale factors
	xScale := float64(imgWidth) / float64(targetWidth)
	yScale := float64(imgHeight) / float64(targetHeight)
	
	// Process each row
	for row := 0; row < charHeight; row++ {
		var rowStr strings.Builder
		
		for col := 0; col < charWidth; col++ {
			// Sample two pixels for half-block (upper and lower)
			y1 := int(float64(row*2) * yScale)
			y2 := int(float64(row*2+1) * yScale)
			x := int(float64(col) * xScale)
			
			// Bounds checking
			if x >= imgWidth {
				x = imgWidth - 1
			}
			if y1 >= imgHeight {
				y1 = imgHeight - 1
			}
			if y2 >= imgHeight {
				y2 = imgHeight - 1
			}
			
			// Get colors
			c1 := img.At(x, y1)
			c2 := img.At(x, y2)
			
			// Convert to RGB
			r1, g1, b1, _ := c1.RGBA()
			r2, g2, b2, _ := c2.RGBA()
			
			// Convert to 8-bit color
			r1, g1, b1 = r1>>8, g1>>8, b1>>8
			r2, g2, b2 = r2>>8, g2>>8, b2>>8
			
			// Generate ANSI escape sequences
			// Upper half block with foreground color
			// Lower half block with background color
			rowStr.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▀\x1b[0m", r1, g1, b1, r2, g2, b2))
		}
		
		preview.Rows[row] = rowStr.String()
	}
	
	return preview
}

// readSVGInfo reads SVG metadata and generates preview
func readSVGInfo(path string, fileSize int64, maxPreviewHeight, maxPreviewWidth int) (*ImageInfo, error) {
	// Read SVG file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	// Try to extract dimensions from SVG content
	svgStr := string(data)
	width, height := extractSVGDimensions(svgStr)
	
	info := &ImageInfo{
		Format: "svg",
		Width:  width,
		Height: height,
		Size:   fileSize,
	}
	
	// Generate preview if requested
	if maxPreviewHeight > 15 { // Only generate preview if we have reasonable space
		// Calculate available space
		availableHeight := maxPreviewHeight - 4 // Same as regular images
		if availableHeight > 0 {
			// Use the actual available width minus padding
			availableWidth := maxPreviewWidth - 4
			if availableWidth < 20 {
				availableWidth = 20
			}
			
			// Calculate render size to fit in preview area
			renderWidth := width
			renderHeight := height
			
			// Scale down if needed to fit preview constraints
			if renderHeight > availableHeight*10 || renderWidth > availableWidth*10 {
				aspectRatio := float64(width) / float64(height)
				if aspectRatio > float64(availableWidth)/float64(availableHeight) {
					renderWidth = availableWidth * 10
					renderHeight = int(float64(renderWidth) / aspectRatio)
				} else {
					renderHeight = availableHeight * 10
					renderWidth = int(float64(renderHeight) * aspectRatio)
				}
			}
			
			// Use oksvg implementation
			icon, parseErr := oksvg.ReadIconStream(bytes.NewReader(data))
			if parseErr != nil {
				info.Preview = &ImagePreview{
					Width:  1,
					Height: 1,
					Rows:   []string{fmt.Sprintf("[SVG parsing error: %v]", parseErr)},
				}
			} else {
				// Try oksvg rendering
				img, renderErr := rasterizeSVG(icon, renderWidth, renderHeight)
				if renderErr != nil {
					info.Preview = &ImagePreview{
						Width:  1,
						Height: 1,
						Rows:   []string{fmt.Sprintf("[SVG rendering error: %v]", renderErr)},
					}
				} else {
					info.Preview = generateImagePreview(img, availableWidth, availableHeight)
				}
			}
		}
	}
	
	return info, nil
}

// extractSVGDimensions tries to extract width and height from SVG content
func extractSVGDimensions(svgContent string) (int, int) {
	width, height := 256, 256 // defaults
	
	// Try to extract width
	if widthMatch := strings.Index(svgContent, `width="`); widthMatch != -1 {
		widthStart := widthMatch + 7
		widthEnd := strings.Index(svgContent[widthStart:], `"`)
		if widthEnd != -1 {
			if w, err := strconv.Atoi(svgContent[widthStart:widthStart+widthEnd]); err == nil {
				width = w
			}
		}
	}
	
	// Try to extract height
	if heightMatch := strings.Index(svgContent, `height="`); heightMatch != -1 {
		heightStart := heightMatch + 8
		heightEnd := strings.Index(svgContent[heightStart:], `"`)
		if heightEnd != -1 {
			if h, err := strconv.Atoi(svgContent[heightStart:heightStart+heightEnd]); err == nil {
				height = h
			}
		}
	}
	
	// Try viewBox if width/height not found
	if viewBoxMatch := strings.Index(svgContent, `viewBox="`); viewBoxMatch != -1 {
		viewBoxStart := viewBoxMatch + 9
		viewBoxEnd := strings.Index(svgContent[viewBoxStart:], `"`)
		if viewBoxEnd != -1 {
			viewBox := svgContent[viewBoxStart:viewBoxStart+viewBoxEnd]
			parts := strings.Fields(viewBox)
			if len(parts) >= 4 {
				if w, err := strconv.ParseFloat(parts[2], 64); err == nil {
					width = int(w)
				}
				if h, err := strconv.ParseFloat(parts[3], 64); err == nil {
					height = int(h)
				}
			}
		}
	}
	
	return width, height
}

// rasterizeSVG converts an SVG icon to a raster image using oksvg
func rasterizeSVG(icon *oksvg.SvgIcon, width, height int) (image.Image, error) {
	// Limit render size to prevent memory issues
	maxSize := 1024
	if width > maxSize || height > maxSize {
		scale := float64(maxSize) / float64(max(width, height))
		width = int(float64(width) * scale)
		height = int(float64(height) * scale)
	}
	
	// Create a new RGBA image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Set icon size to fit the canvas
	icon.SetTarget(0, 0, float64(width), float64(height))
	
	// Create a rasterizer
	scanner := rasterx.NewScannerGV(width, height, img, img.Bounds())
	raster := rasterx.NewDasher(width, height, scanner)
	
	// Try to draw the SVG
	defer func() {
		if r := recover(); r != nil {
			// Recover from panics in SVG rendering
			err := fmt.Errorf("SVG rendering panic: %v", r)
			fmt.Println(err)
		}
	}()
	
	// Draw the icon
	icon.Draw(raster, 1.0)
	
	return img, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ReadDatabaseContent reads information from a database file
func ReadDatabaseContent(path string) (*DatabaseInfo, error) {
	return readDatabaseInfo(path)
}

// readDatabaseInfo reads information from a database file
func readDatabaseInfo(path string) (*DatabaseInfo, error) {
	// Try to open as SQLite database
	db, err := sql.Open("sqlite3", path+"?mode=ro") // Read-only mode
	if err != nil {
		return &DatabaseInfo{Error: err.Error()}, nil
	}
	defer db.Close()
	
	// Test connection
	if err := db.Ping(); err != nil {
		return &DatabaseInfo{Error: "Not a valid SQLite database: " + err.Error()}, nil
	}
	
	dbInfo := &DatabaseInfo{
		Format: "SQLite",
	}
	
	// Get database file size
	if stat, err := os.Stat(path); err == nil {
		dbInfo.FileSize = stat.Size()
	}
	
	// Get SQLite version
	if version, err := getSQLiteVersion(db); err == nil {
		dbInfo.Version = version
	}
	
	// Get all tables
	tables, err := getAllTables(db)
	if err != nil {
		dbInfo.Error = err.Error()
		return dbInfo, nil
	}
	
	dbInfo.Tables = tables
	dbInfo.TableCount = len(tables)
	
	// Generate schema dump
	if schema, err := generateSchema(db, tables); err == nil {
		dbInfo.Schema = schema
	}
	
	return dbInfo, nil
}

// getSQLiteVersion gets the SQLite version
func getSQLiteVersion(db *sql.DB) (string, error) {
	var version string
	err := db.QueryRow("SELECT sqlite_version()").Scan(&version)
	return version, err
}

// getAllTables gets information about all tables in the database
func getAllTables(db *sql.DB) ([]TableInfo, error) {
	// Query sqlite_master for all tables
	rows, err := db.Query(`
		SELECT name, sql FROM sqlite_master 
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tables []TableInfo
	for rows.Next() {
		var tableName, tableSQL string
		if err := rows.Scan(&tableName, &tableSQL); err != nil {
			continue
		}
		
		table := TableInfo{
			Name:   tableName,
			Schema: tableSQL,
		}
		
		// Get row count
		if count, err := getTableRowCount(db, tableName); err == nil {
			table.RowCount = count
		}
		
		// Get column info
		if columns, err := getTableColumns(db, tableName); err == nil {
			table.Columns = columns
		}
		
		// Get sample data (first 5 rows)
		if sample, err := getTableSample(db, tableName, 5); err == nil {
			table.Sample = sample
		}
		
		tables = append(tables, table)
	}
	
	return tables, nil
}

// getTableRowCount gets the number of rows in a table
func getTableRowCount(db *sql.DB, tableName string) (int64, error) {
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", tableName)
	err := db.QueryRow(query).Scan(&count)
	return count, err
}

// getTableColumns gets information about table columns
func getTableColumns(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	query := fmt.Sprintf("PRAGMA table_info(\"%s\")", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var columns []ColumnInfo
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString
		
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}
		
		columns = append(columns, ColumnInfo{
			Name:    name,
			Type:    dataType,
			NotNull: notNull == 1,
			PK:      pk == 1,
		})
	}
	
	return columns, nil
}

// getTableSample gets sample data from a table
func getTableSample(db *sql.DB, tableName string, limit int) ([]map[string]any, error) {
	query := fmt.Sprintf("SELECT * FROM \"%s\" LIMIT %d", tableName, limit)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	
	var result []map[string]any
	for rows.Next() {
		// Create slice to hold values
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		
		// Create map from column names to values
		row := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				// Convert byte slices to strings for display
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		
		result = append(result, row)
	}
	
	return result, nil
}

// generateSchema generates a schema dump for the database
func generateSchema(db *sql.DB, tables []TableInfo) (string, error) {
	var schema strings.Builder
	
	schema.WriteString("-- SQLite Database Schema\n\n")
	
	for _, table := range tables {
		if table.Schema != "" {
			schema.WriteString(table.Schema)
			schema.WriteString(";\n\n")
		}
	}
	
	return schema.String(), nil
}

// ReadTableData reads paginated data from a specific table
func ReadTableData(dbPath, tableName string, offset, limit int) ([]map[string]any, error) {
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	
	if err := db.Ping(); err != nil {
		return nil, err
	}
	
	// Build query with pagination
	query := fmt.Sprintf("SELECT * FROM \"%s\" LIMIT %d OFFSET %d", tableName, limit, offset)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	
	var result []map[string]any
	for rows.Next() {
		// Create slice to hold values
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		
		// Create map from column names to values
		row := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				// Convert byte slices to strings for display
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		
		result = append(result, row)
	}
	
	return result, nil
}