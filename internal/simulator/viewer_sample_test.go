package simulator

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTextDetectionCompletesUTF8AtTheSampleBoundary(t *testing.T) {
	for _, character := range []string{"é", "世", "\U00010400"} {
		for split := 1; split < len(character); split++ {
			data := strings.Repeat("a", 512-split) + character + "\n"
			path := filepath.Join(t.TempDir(), "text-without-extension")
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			content, err := ReadFileContent(path, 0, 10, 80)
			if err != nil || content.Type != FileTypeText || len(content.Lines) != 1 || content.Lines[0] != strings.TrimSuffix(data, "\n") {
				t.Fatalf("UTF-8 split after byte %d: content=%+v, err=%v", split, content, err)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "incomplete")
	if err := os.WriteFile(path, []byte(strings.Repeat("a", 511)+"\xc3"), 0600); err != nil {
		t.Fatal(err)
	}
	if DetectFileType(path) != FileTypeBinary {
		t.Fatal("incomplete UTF-8 at EOF was accepted as text")
	}
}

func TestBMPAndTIFFFilesHaveImagePreviews(t *testing.T) {
	bmp := make([]byte, 58)
	copy(bmp, "BM")
	for offset, value := range map[int]uint32{2: 58, 10: 54, 14: 40, 18: 1, 22: 1} {
		binary.LittleEndian.PutUint32(bmp[offset:], value)
	}
	binary.LittleEndian.PutUint16(bmp[26:], 1)
	binary.LittleEndian.PutUint16(bmp[28:], 24)

	// A one-pixel, uncompressed grayscale TIFF; no decoder imports in tests.
	tiff := make([]byte, 123)
	copy(tiff, "II")
	binary.LittleEndian.PutUint16(tiff[2:], 42)
	binary.LittleEndian.PutUint32(tiff[4:], 8)
	tags := [][3]uint32{
		{256, 4, 1}, {257, 4, 1}, {258, 3, 8}, {259, 3, 1},
		{262, 3, 1}, {273, 4, 122}, {277, 3, 1}, {278, 4, 1}, {279, 4, 1},
	}
	binary.LittleEndian.PutUint16(tiff[8:], uint16(len(tags)))
	for i, tag := range tags {
		entry := tiff[10+i*12:]
		binary.LittleEndian.PutUint16(entry, uint16(tag[0]))
		binary.LittleEndian.PutUint16(entry[2:], uint16(tag[1]))
		binary.LittleEndian.PutUint32(entry[4:], 1)
		binary.LittleEndian.PutUint32(entry[8:], tag[2])
	}
	for name, data := range map[string][]byte{"black.bmp": bmp, "black.tiff": tiff} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			content, err := ReadFileContent(path, 0, 24, 80)
			if err != nil {
				t.Fatal(err)
			}
			if content.Type != FileTypeImage || content.ImageInfo == nil || content.ImageInfo.Preview == nil {
				t.Fatalf("expected image preview, got %+v", content)
			}
			if content.ImageInfo.Width != 1 || content.ImageInfo.Height != 1 {
				t.Fatalf("unexpected dimensions: %+v", content.ImageInfo)
			}
		})
	}
}
