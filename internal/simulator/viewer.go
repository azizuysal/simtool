package simulator

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	_ "golang.org/x/image/webp" // Add WebP support
)

// FileType represents the type of file for viewing
type FileType int

const (
	FileTypeText FileType = iota
	FileTypeImage
	FileTypeBinary
	FileTypeArchive
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
	
	// For all files (including those with text-like extensions), check content
	file, err := os.Open(path)
	if err != nil {
		return FileTypeBinary
	}
	defer file.Close()
	
	// Read first 512 bytes to check if it's text
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return FileTypeBinary
	}
	
	// Check if the content is valid UTF-8 and mostly printable
	if isTextContent(buffer[:n]) {
		// Common text file extensions
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
		}
		
		// If it has a known text extension or no extension, treat as text
		if textExts[ext] || ext == "" {
			return FileTypeText
		}
	}
	
	return FileTypeBinary
}

// isTextContent checks if the content appears to be text
func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	
	// Check for binary plist magic bytes
	if len(data) >= 6 && string(data[:6]) == "bplist" {
		return false
	}
	
	// Check if it's valid UTF-8
	if !utf8.Valid(data) {
		return false
	}
	
	// Count printable vs non-printable characters
	printable := 0
	for _, b := range data {
		// Allow printable ASCII, newlines, tabs, carriage returns
		if (b >= 32 && b <= 126) || b == '\n' || b == '\t' || b == '\r' {
			printable++
		}
	}
	
	// If more than 90% of characters are printable, consider it text
	return float64(printable)/float64(len(data)) > 0.9
}

// ReadFileContent reads file content based on its type
func ReadFileContent(path string, startLine, maxLines int) (*FileContent, error) {
	fileType := DetectFileType(path)
	
	content := &FileContent{
		Type: fileType,
	}
	
	switch fileType {
	case FileTypeText:
		lines, totalLines, err := readTextFile(path, startLine, maxLines)
		content.Lines = lines
		content.TotalLines = totalLines
		content.Error = err
		
	case FileTypeImage:
		info, err := readImageInfo(path, maxLines) // Pass maxLines for preview size
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
	}
	
	return content, content.Error
}

// readTextFile reads a text file with pagination support
func readTextFile(path string, startLine, maxLines int) ([]string, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
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
		return lines, totalLines, err
	}
	
	return lines, totalLines, nil
}

// readImageInfo reads image metadata and generates preview
func readImageInfo(path string, maxPreviewHeight int) (*ImageInfo, error) {
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
			// Reserve space: ~8 lines for metadata, 4 for padding/borders
			availableHeight := maxPreviewHeight - 12
			if availableHeight > 0 {
				// Width is typically 2-3x height in terminals
				maxWidth := availableHeight * 3
				info.Preview = generateImagePreview(img, maxWidth, availableHeight)
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
	termFormatter = formatters.Get("terminal16m")
	chromaStyle   = styles.Get("monokai")
)

// GetSyntaxHighlightedLine returns a syntax highlighted version of a line
// This is a simple implementation - could be enhanced with a proper syntax highlighting library
func GetSyntaxHighlightedLine(line string, fileExt string) string {
	// Quick return for empty lines
	if strings.TrimSpace(line) == "" {
		return line
	}
	
	// Get or create lexer for this file extension
	lexer := getLexerForExtension(fileExt)
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
	err = termFormatter.Format(&buf, chromaStyle, iterator)
	if err != nil {
		return line
	}
	
	return strings.TrimRight(buf.String(), "\n")
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
		case ".yml":
			lexer = lexers.Get("yaml")
		case ".tsx":
			lexer = lexers.Get("typescript")
		case ".jsx":
			lexer = lexers.Get("javascript")
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