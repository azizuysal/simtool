package simulator

import (
	"bufio"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// FileType represents the type of file for viewing
type FileType int

const (
	FileTypeText FileType = iota
	FileTypeImage
	FileTypeBinary
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
	Error       error
}

// ImageInfo contains metadata about an image file
type ImageInfo struct {
	Format string
	Width  int
	Height int
	Size   int64
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
		info, err := readImageInfo(path)
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
	
	// Set a reasonable buffer size for long lines
	const maxScanTokenSize = 256 * 1024 // 256KB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)
	
	for scanner.Scan() {
		totalLines++
		if currentLine >= startLine && len(lines) < maxLines {
			line := scanner.Text()
			// Truncate very long lines for display
			if len(line) > 500 {
				line = line[:500] + "..."
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

// readImageInfo reads image metadata
func readImageInfo(path string) (*ImageInfo, error) {
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
	
	return &ImageInfo{
		Format: format,
		Width:  config.Width,
		Height: config.Height,
		Size:   stat.Size(),
	}, nil
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

// GetSyntaxHighlightedLine returns a syntax highlighted version of a line
// This is a simple implementation - could be enhanced with a proper syntax highlighting library
func GetSyntaxHighlightedLine(line string, fileExt string) string {
	// For now, just return the line as-is
	// In a full implementation, this would apply syntax highlighting based on file type
	return line
}