package simulator

import (
	"archive/zip"
	"fmt"
)

// readArchiveInfo reads information about an archive file
func readArchiveInfo(path string) (*ArchiveInfo, error) {
	// Open the ZIP file
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() { _ = reader.Close() }()

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
