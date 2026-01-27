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

	// First pass: count book files in ZIP for accurate progress
	var bookFiles []*zip.File
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		lowerName := strings.ToLower(file.Name)
		if strings.HasSuffix(lowerName, ".fb2") || strings.HasSuffix(lowerName, ".epub") {
			bookFiles = append(bookFiles, file)
		}
	}

	totalBooksInZip := len(bookFiles)
	log.Printf("Processing ZIP file: %s (%d books inside)", fileInfo.FileName, totalBooksInZip)

	// Report initial ZIP progress
	si.reportZipProgress(si.currentFileIndex+1, si.totalBooks, 0, totalBooksInZip, fileInfo.FileName,
		fmt.Sprintf("Starting %s (0/%d books)...", fileInfo.FileName, totalBooksInZip))

	// Second pass: process each book file
	for i, file := range bookFiles {
		// Check for cancellation before each book
		select {
		case <-ctx.Done():
			// Commit any remaining books in current batch before exiting
			if len(si.currentBatch) > 0 {
				if err := si.commitBatch(); err != nil {
					log.Printf("Warning: failed to commit final batch on cancellation: %v", err)
				}
			}
			log.Printf("ZIP processing canceled after %d/%d books in %s", i, totalBooksInZip, fileInfo.FileName)
			return ctx.Err()
		default:
		}

		fileName := file.Name

		// Parse the book from ZIP
		scannedBook, err := si.parseBookFromZip(fileInfo.Path, file)
		if err != nil {
			log.Printf("Warning: failed to parse %s in %s: %v", fileName, fileInfo.FileName, err)
			si.skippedBooks++

			// Report progress every 100 books
			if (i+1)%100 == 0 {
				si.reportZipProgress(si.currentFileIndex+1, si.totalBooks, i+1, totalBooksInZip, fileInfo.FileName,
					fmt.Sprintf("Processing %s: %d/%d books (imported %d, skipped %d)...", fileInfo.FileName, i+1, totalBooksInZip, si.importedBooks, si.skippedBooks))
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
			si.reportZipProgress(si.currentFileIndex+1, si.totalBooks, i+1, totalBooksInZip, fileInfo.FileName,
				fmt.Sprintf("Processing %s: %d/%d books (imported %d, skipped %d)...", fileInfo.FileName, i+1, totalBooksInZip, si.importedBooks, si.skippedBooks))
		}
	}

	// Commit any remaining books in final partial batch
	if len(si.currentBatch) > 0 {
		if err := si.commitBatch(); err != nil {
			log.Printf("Warning: failed to commit final batch: %v", err)
		}
	}

	log.Printf("Completed processing ZIP: %s (%d books processed, %d imported, %d skipped)",
		fileInfo.FileName, totalBooksInZip, si.importedBooks, si.skippedBooks)

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
