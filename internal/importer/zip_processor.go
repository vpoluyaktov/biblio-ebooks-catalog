package importer

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"biblio-opds-server/internal/parser"
)

// processZipFile processes all books inside a ZIP archive
func (si *StreamingImporter) processZipFile(ctx context.Context, fileInfo *FileInfo) error {
	reader, err := zip.OpenReader(fileInfo.Path)
	if err != nil {
		return fmt.Errorf("failed to open ZIP: %w", err)
	}
	defer reader.Close()

	log.Printf("Processing ZIP file: %s (%d files inside)", fileInfo.FileName, len(reader.File))

	booksProcessed := 0

	// Process each file in the ZIP
	for _, file := range reader.File {
		// Check for cancellation before each book
		select {
		case <-ctx.Done():
			// Commit any remaining books in current batch before exiting
			if len(si.currentBatch) > 0 {
				if err := si.commitBatch(); err != nil {
					log.Printf("Warning: failed to commit final batch on cancellation: %v", err)
				}
			}
			log.Printf("ZIP processing canceled after %d books in %s", booksProcessed, fileInfo.FileName)
			return ctx.Err()
		default:
		}

		if file.FileInfo().IsDir() {
			continue
		}

		fileName := file.Name
		lowerName := strings.ToLower(fileName)

		// Check if it's a book file
		if !strings.HasSuffix(lowerName, ".fb2") && !strings.HasSuffix(lowerName, ".epub") {
			continue
		}

		booksProcessed++

		// Parse the book from ZIP
		scannedBook, err := si.parseBookFromZip(fileInfo.Path, file)
		if err != nil {
			log.Printf("Warning: failed to parse %s in %s: %v", fileName, fileInfo.FileName, err)
			si.skippedBooks++

			// Report progress every 100 books (including skipped)
			if booksProcessed%100 == 0 {
				si.reportProgress(si.importedBooks+si.skippedBooks, si.totalBooks,
					fmt.Sprintf("Processing %s: imported %d books, skipped %d...", fileInfo.FileName, si.importedBooks, si.skippedBooks))
			}
			continue
		}

		// Add to current batch
		si.currentBatch = append(si.currentBatch, scannedBook)

		// Commit batch if it reaches batch size
		if len(si.currentBatch) >= si.batchSize {
			if err := si.commitBatch(); err != nil {
				log.Printf("Warning: batch commit error: %v", err)
			}
			// Report progress after each batch commit
			si.reportProgress(si.importedBooks+si.skippedBooks, si.totalBooks,
				fmt.Sprintf("Processing %s: imported %d books, skipped %d...", fileInfo.FileName, si.importedBooks, si.skippedBooks))
		}
	}

	// Commit any remaining books in final partial batch
	if len(si.currentBatch) > 0 {
		if err := si.commitBatch(); err != nil {
			log.Printf("Warning: failed to commit final batch: %v", err)
		}
	}

	log.Printf("Completed processing ZIP: %s (%d books processed, %d imported, %d skipped)",
		fileInfo.FileName, booksProcessed, si.importedBooks, si.skippedBooks)

	return nil
}

// parseBookFromZip parses a single book from a ZIP file
func (si *StreamingImporter) parseBookFromZip(zipPath string, file *zip.File) (*ScannedBook, error) {
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

	// Determine format
	fileName := file.Name
	lowerName := strings.ToLower(fileName)
	var format string
	if strings.HasSuffix(lowerName, ".fb2") {
		format = "fb2"
	} else if strings.HasSuffix(lowerName, ".epub") {
		format = "epub"
	} else {
		return nil, fmt.Errorf("unknown format: %s", fileName)
	}

	// Parse metadata
	metadata, err := parser.ParseFromBytes(format, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Create ScannedBook
	scannedBook := &ScannedBook{
		FilePath:  zipPath,
		RelPath:   zipPath,
		FileName:  fileName,
		Format:    format,
		Size:      int64(file.UncompressedSize64),
		Metadata:  metadata,
		Archive:   zipPath,
		FileInZip: fileName,
	}

	return scannedBook, nil
}
