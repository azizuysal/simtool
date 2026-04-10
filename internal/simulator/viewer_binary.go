package simulator

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// readBinaryFile reads a chunk of binary file
func readBinaryFile(path string, offset int64, size int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	// Seek to offset
	_, err = file.Seek(offset, 0)
	if err != nil {
		return nil, err
	}

	// Read chunk
	data := make([]byte, size)
	n, err := file.Read(data)
	if err != nil && !errors.Is(err, io.EOF) {
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
