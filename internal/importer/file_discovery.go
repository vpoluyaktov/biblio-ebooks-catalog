package importer

import (
	"archive/zip"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// FileDiscovery handles the file discovery phase of import
type FileDiscovery struct {
	libraryPath string
	progress    ProgressCallback
}

// NewFileDiscovery creates a new file discovery instance
func NewFileDiscovery(libraryPath string) *FileDiscovery {
	return &FileDiscovery{
		libraryPath: libraryPath,
	}
}

// SetProgressCallback sets the progress callback function
func (fd *FileDiscovery) SetProgressCallback(cb ProgressCallback) {
	fd.progress = cb
}

// DiscoverFiles performs fast file discovery without parsing
// Returns a list of FileInfo structs representing all books found
func (fd *FileDiscovery) DiscoverFiles() ([]*FileInfo, error) {
	var files []*FileInfo

	log.Printf("Starting file discovery in: %s", fd.libraryPath)

	// Walk directory tree
	err := filepath.Walk(fd.libraryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Warning: error accessing path %s: %v", path, err)
			return nil // Continue walking
		}

		if info.IsDir() {
			return nil // Continue into directory
		}

		// Get relative path
		relPath, err := filepath.Rel(fd.libraryPath, path)
		if err != nil {
			relPath = path
		}

		fileName := info.Name()
		fileSize := info.Size()

		// Determine file type
		lowerName := strings.ToLower(fileName)

		if strings.HasSuffix(lowerName, ".fb2") {
			// Single FB2 file
			files = append(files, &FileInfo{
				Path:     path,
				RelPath:  relPath,
				FileName: fileName,
				Type:     FileTypeFB2,
				Size:     fileSize,
			})
		} else if strings.HasSuffix(lowerName, ".epub") {
			// EPUB file
			files = append(files, &FileInfo{
				Path:     path,
				RelPath:  relPath,
				FileName: fileName,
				Type:     FileTypeEPUB,
				Size:     fileSize,
			})
		} else if strings.HasSuffix(lowerName, ".zip") {
			// ZIP archive - need to list contents
			zipFiles, err := fd.listZipContents(path, relPath, fileSize)
			if err != nil {
				log.Printf("Warning: failed to list ZIP contents for %s: %v", path, err)
				return nil // Continue walking
			}
			files = append(files, zipFiles...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	log.Printf("File discovery complete: found %d files", len(files))
	if fd.progress != nil {
		fd.progress(0, len(files), fmt.Sprintf("Found %d files, starting import...", len(files)))
	}

	return files, nil
}

// listZipContents lists all book files inside a ZIP archive
func (fd *FileDiscovery) listZipContents(zipPath, relPath string, zipSize int64) ([]*FileInfo, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP: %w", err)
	}
	defer reader.Close()

	var files []*FileInfo

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		fileName := file.Name
		lowerName := strings.ToLower(fileName)

		// Check if it's a book file
		if strings.HasSuffix(lowerName, ".fb2") || strings.HasSuffix(lowerName, ".epub") {
			offset, err := file.DataOffset()
			if err != nil {
				log.Printf("Warning: failed to get offset for %s in %s: %v", fileName, zipPath, err)
				offset = 0 // Use 0 as fallback
			}

			files = append(files, &FileInfo{
				Path:        zipPath,
				RelPath:     relPath,
				FileName:    fileName,
				Type:        FileTypeInZip,
				Size:        zipSize, // Size of ZIP file
				ZipPath:     zipPath,
				FileInZip:   fileName,
				OffsetInZip: offset,
				SizeInZip:   int64(file.UncompressedSize64),
			})
		}
	}

	return files, nil
}
