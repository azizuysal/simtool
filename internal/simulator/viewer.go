package simulator

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
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

// Constants shared across the viewer subsystem. Exported ones are
// referenced from the tui package (e.g. for converting byte offsets
// to hex-dump line offsets when paginating).
const (
	// BinaryChunkSize is the number of bytes read per binary file fetch.
	BinaryChunkSize = 8192
	// HexBytesPerLine is the number of bytes shown per hex dump row.
	HexBytesPerLine = 16
)

// FileContent represents the content of a file prepared for viewing
type FileContent struct {
	Type          FileType
	Lines         []string // For text files
	TotalLines    int      // Total number of lines in the file
	ImageInfo     *ImageInfo
	BinaryData    []byte        // For hex view (current chunk)
	BinaryOffset  int64         // Offset of the current chunk in the file
	TotalSize     int64         // Total size of the file (for binary files)
	ArchiveInfo   *ArchiveInfo  // For archive files
	DatabaseInfo  *DatabaseInfo // For database files
	IsBinaryPlist bool          // Whether this was converted from binary plist
	DetectedLang  string        // Detected language for syntax highlighting (e.g., "html")
	Error         error
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
	Name           string
	Size           int64
	CompressedSize int64
	ModTime        time.Time
	IsDir          bool
}

// DatabaseInfo contains information about a database file
type DatabaseInfo struct {
	Format     string      `json:"format"`  // "SQLite", "MySQL", etc.
	Version    string      `json:"version"` // Database version
	FileSize   int64       `json:"file_size"`
	TableCount int         `json:"table_count"`
	Tables     []TableInfo `json:"tables"`
	Schema     string      `json:"schema"` // Full schema dump
	Error      string      `json:"error,omitempty"`
}

// TableInfo represents information about a database table
type TableInfo struct {
	Name     string           `json:"name"`
	RowCount int64            `json:"row_count"`
	Schema   string           `json:"schema"`
	Columns  []ColumnInfo     `json:"columns"`
	Sample   []map[string]any `json:"sample,omitempty"` // First few rows
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
	defer func() { _ = file.Close() }()

	// Read first 512 bytes to check content
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return FileTypeBinary
	}

	// Check for SQLite magic header "SQLite format 3\000"
	if n >= 16 && string(buffer[:15]) == "SQLite format 3" {
		return FileTypeDatabase
	}

	// Check for image file signatures
	imageSignatures := [][]byte{
		[]byte("\x89PNG\r\n\x1a\n"),        // PNG
		[]byte("\xFF\xD8\xFF"),             // JPEG
		[]byte("GIF87a"), []byte("GIF89a"), // GIF
		[]byte("RIFF"),                             // RIFF (might be WebP)
		[]byte("BM"),                               // BMP
		[]byte("MM\x00\x2A"), []byte("II\x2A\x00"), // TIFF
		[]byte("\x00\x00\x01\x00"), // ICO
		[]byte("\x00\x00\x02\x00"), // CUR
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
		[]byte("\x1F\x8B"),     // GZIP
		[]byte("BZh"),          // BZIP2
		[]byte("\xFD7zXZ\x00"), // XZ
		[]byte("Rar!"),         // RAR
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
		[]byte("bplist"),                   // Binary plist
		[]byte("\x89PNG\r\n\x1a\n"),        // PNG
		[]byte("\xFF\xD8\xFF"),             // JPEG
		[]byte("GIF87a"), []byte("GIF89a"), // GIF
		[]byte("RIFF"),                             // RIFF (WebP, WAV, AVI)
		[]byte("\x00\x00\x01\x00"),                 // ICO
		[]byte("\x00\x00\x02\x00"),                 // CUR
		[]byte("BM"),                               // BMP
		[]byte("MM\x00\x2A"), []byte("II\x2A\x00"), // TIFF
		[]byte("PK\x03\x04"), []byte("PK\x05\x06"), // ZIP and variants
		[]byte("\xCA\xFE\xBA\xBE"),                 // Mach-O binary
		[]byte("\xCE\xFA\xED\xFE"),                 // Mach-O binary (32-bit)
		[]byte("\xCF\xFA\xED\xFE"),                 // Mach-O binary (64-bit)
		[]byte("\xFE\xED\xFA\xCE"),                 // Mach-O binary (big-endian)
		[]byte("\xFE\xED\xFA\xCF"),                 // Mach-O binary (64-bit big-endian)
		[]byte("SQLite format 3"),                  // SQLite database
		[]byte("\x1F\x8B"),                         // GZIP
		[]byte("BZh"),                              // BZIP2
		[]byte("\xFD7zXZ\x00"),                     // XZ
		[]byte("Rar!"),                             // RAR
		[]byte("\x50\x4B"),                         // PKZip
		[]byte("\x7FELF"),                          // ELF binary
		[]byte("%PDF-"),                            // PDF
		[]byte("\x25\x50\x44\x46\x2D"),             // PDF (hex)
		[]byte("\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1"), // Microsoft Office (doc, xls, ppt)
		[]byte("\x50\x4B\x03\x04"),                 // Microsoft Office (docx, xlsx, pptx)
		[]byte("\x4F\x67\x67\x53"),                 // OGG
		[]byte("\x38\x42\x50\x53"),                 // PSD
		[]byte("\x52\x49\x46\x46"),                 // WAV
		[]byte("\x00\x00\x00\x0C\x6A\x50\x20\x20"), // JPEG 2000
		[]byte("\x1A\x45\xDF\xA3"),                 // MKV, WebM
		[]byte("\x00\x00\x00\x14\x66\x74\x79\x70"), // MP4, M4V, M4A
		[]byte("\x49\x44\x33"),                     // MP3
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

			// Offset is expressed in hex-dump lines on entry.
			offset := int64(startLine * HexBytesPerLine)

			readSize := BinaryChunkSize

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

		// Offset is expressed in hex-dump lines on entry.
		offset := int64(startLine * HexBytesPerLine)

		readSize := BinaryChunkSize

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
