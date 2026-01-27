package importer

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"

	"biblio-opds-server/internal/db"
	"biblio-opds-server/internal/parser"
)

// StreamingImporter imports books one-by-one with frequent cancellation checks
type StreamingImporter struct {
	db              *db.DB
	libraryID       int64
	libraryPath     string
	firstAuthorOnly bool
	progress        ProgressCallback

	// Import statistics
	totalBooks    int
	importedBooks int
	skippedBooks  int

	// Batch management
	batchSize    int
	currentBatch []*ScannedBook

	// Genre codes
	genreCodes map[string]int
}

// NewStreamingImporter creates a new streaming importer
func NewStreamingImporter(database *db.DB, libraryID int64, libraryPath string, firstAuthorOnly bool) *StreamingImporter {
	return &StreamingImporter{
		db:              database,
		libraryID:       libraryID,
		libraryPath:     libraryPath,
		firstAuthorOnly: firstAuthorOnly,
		batchSize:       100,
		currentBatch:    make([]*ScannedBook, 0, 100),
		genreCodes:      make(map[string]int),
	}
}

// SetProgressCallback sets the progress callback function
func (si *StreamingImporter) SetProgressCallback(cb ProgressCallback) {
	si.progress = cb
}

// LoadGenreCodes loads genre codes from file
func (si *StreamingImporter) LoadGenreCodes() error {
	imp := New(si.db)
	if err := imp.loadGenreCodes(); err != nil {
		return fmt.Errorf("failed to load genre codes: %w", err)
	}
	si.genreCodes = imp.genreCodes
	return nil
}

// ImportFiles imports a list of discovered files with streaming
func (si *StreamingImporter) ImportFiles(ctx context.Context, files []*FileInfo) error {
	si.totalBooks = len(files)

	log.Printf("Starting streaming import of %d files to library %d", si.totalBooks, si.libraryID)
	// Use 0 as total to indicate indeterminate progress (we don't know total book count for ZIPs)
	si.reportProgress(0, 0, "Starting import...")

	for i, fileInfo := range files {
		// Check for cancellation before each file
		select {
		case <-ctx.Done():
			// Commit any remaining books in current batch
			if len(si.currentBatch) > 0 {
				if err := si.commitBatch(); err != nil {
					log.Printf("Warning: failed to commit final batch: %v", err)
				}
			}
			log.Printf("Import canceled after %d books imported, %d skipped", si.importedBooks, si.skippedBooks)
			return ctx.Err()
		default:
		}

		// Handle ZIP files specially - process all books inside
		if fileInfo.Type == FileTypeZIP {
			if err := si.processZipFile(ctx, fileInfo); err != nil {
				log.Printf("Warning: failed to process ZIP %s: %v", fileInfo.FileName, err)
			}
			// Use 0 as total for indeterminate progress
			si.reportProgress(si.importedBooks+si.skippedBooks, 0, fmt.Sprintf("Processed %d/%d files, imported %d books, skipped %d...", i+1, si.totalBooks, si.importedBooks, si.skippedBooks))
			continue
		}

		// Parse single file
		scannedBook, err := si.parseFile(fileInfo)
		if err != nil {
			log.Printf("Warning: failed to parse %s: %v", fileInfo.FileName, err)
			si.skippedBooks++
			si.reportProgress(si.importedBooks+si.skippedBooks, 0, fmt.Sprintf("Imported %d books, skipped %d...", si.importedBooks, si.skippedBooks))
			continue
		}

		// Add to current batch
		si.currentBatch = append(si.currentBatch, scannedBook)

		// Commit batch if it reaches batch size
		if len(si.currentBatch) >= si.batchSize {
			if err := si.commitBatch(); err != nil {
				log.Printf("Warning: batch commit error: %v", err)
			}
		}

		// Report progress
		si.reportProgress(si.importedBooks+si.skippedBooks, 0, fmt.Sprintf("Imported %d books, skipped %d...", si.importedBooks, si.skippedBooks))
	}

	// Commit any remaining books
	if len(si.currentBatch) > 0 {
		if err := si.commitBatch(); err != nil {
			return fmt.Errorf("failed to commit final batch: %w", err)
		}
	}

	log.Printf("Import complete: %d books imported, %d skipped", si.importedBooks, si.skippedBooks)
	si.reportProgress(si.totalBooks, si.totalBooks, fmt.Sprintf("Complete! %d books imported, %d skipped", si.importedBooks, si.skippedBooks))

	return nil
}

// parseFile parses a single file and returns a ScannedBook
// For ZIP files, this returns nil as they are handled separately
func (si *StreamingImporter) parseFile(fileInfo *FileInfo) (*ScannedBook, error) {
	var metadata *parser.Metadata
	var err error

	if fileInfo.Type == FileTypeZIP {
		// ZIP files are handled by parseZipFile, not here
		return nil, fmt.Errorf("ZIP files should be processed separately")
	} else if fileInfo.IsInZip() {
		// Extract and parse from ZIP
		metadata, err = si.parseFromZip(fileInfo)
	} else {
		// Parse regular file
		format := fileInfo.GetFormat()
		metadata, err = parser.Parse(format, fileInfo.Path)
	}

	if err != nil {
		return nil, err
	}

	// Create ScannedBook
	scannedBook := &ScannedBook{
		FilePath:  fileInfo.Path,
		RelPath:   fileInfo.RelPath,
		FileName:  fileInfo.FileName,
		Format:    fileInfo.GetFormat(),
		Size:      fileInfo.Size,
		Metadata:  metadata,
		Archive:   fileInfo.ZipPath,
		FileInZip: fileInfo.FileInZip,
	}

	return scannedBook, nil
}

// parseFromZip extracts and parses a file from a ZIP archive
func (si *StreamingImporter) parseFromZip(fileInfo *FileInfo) (*parser.Metadata, error) {
	reader, err := zip.OpenReader(fileInfo.ZipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP: %w", err)
	}
	defer reader.Close()

	// Find the file in the ZIP
	for _, file := range reader.File {
		if file.Name == fileInfo.FileInZip {
			// Open the file
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file in ZIP: %w", err)
			}
			defer rc.Close()

			// Read file contents
			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("failed to read file from ZIP: %w", err)
			}

			// Parse metadata
			format := fileInfo.GetFormat()
			metadata, err := parser.ParseFromBytes(format, data)
			if err != nil {
				return nil, fmt.Errorf("failed to parse metadata: %w", err)
			}

			return metadata, nil
		}
	}

	return nil, fmt.Errorf("file %s not found in ZIP", fileInfo.FileInZip)
}

// commitBatch commits the current batch of books to the database
func (si *StreamingImporter) commitBatch() error {
	if len(si.currentBatch) == 0 {
		return nil
	}

	// Use the existing importBookBatch method
	imp := New(si.db)
	imp.libraryID = si.libraryID
	imp.libraryPath = si.libraryPath
	imp.firstAuthorOnly = si.firstAuthorOnly
	imp.genreCodes = si.genreCodes

	count, err := imp.importBookBatch(si.currentBatch)
	if err != nil {
		return err
	}

	si.importedBooks += count
	si.skippedBooks += (len(si.currentBatch) - count)

	// Clear the batch
	si.currentBatch = si.currentBatch[:0]

	return nil
}

// reportProgress reports the current progress
func (si *StreamingImporter) reportProgress(current, total int, message string) {
	if si.progress != nil {
		si.progress(current, total, message)
	}
}
