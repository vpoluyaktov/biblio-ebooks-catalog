package importer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"biblio-opds-server/internal/db"
	"biblio-opds-server/internal/parser"
)

// ScannedBook represents a book found during directory scanning
type ScannedBook struct {
	FilePath   string
	RelPath    string // Relative path from library root
	FileName   string
	Format     string
	Size       int64
	Metadata   *parser.Metadata
	Archive    string // If file is in a ZIP archive
	FileInZip  string // File name inside ZIP (for fb2.zip)
	ParseError error
}

// ScanProgress represents the current scanning progress
type ScanProgress struct {
	FilesFound   int
	FilesParsed  int
	FilesSkipped int
	CurrentFile  string
	Errors       []string
}

// Scanner handles directory scanning for book files
type Scanner struct {
	libraryPath string
	progress    ProgressCallback
	workers     int

	// Supported extensions
	supportedExts map[string]bool
}

// NewScanner creates a new directory scanner
func NewScanner(libraryPath string, workers int) *Scanner {
	if workers <= 0 {
		workers = 4 // Default to 4 workers
	}

	return &Scanner{
		libraryPath: libraryPath,
		workers:     workers,
		supportedExts: map[string]bool{
			".epub":    true,
			".fb2":     true,
			".fb2.zip": true,
			".zip":     true, // For FB2 archives
		},
	}
}

// SetProgressCallback sets the progress callback function
func (s *Scanner) SetProgressCallback(cb ProgressCallback) {
	s.progress = cb
}

// ScanDirectory scans the library directory and returns all found books
func (s *Scanner) ScanDirectory() ([]*ScannedBook, error) {
	return s.ScanDirectoryWithContext(context.Background())
}

// ScanDirectoryWithContext scans the library directory with cancellation support
func (s *Scanner) ScanDirectoryWithContext(ctx context.Context) ([]*ScannedBook, error) {
	// First pass: find all book files
	filePaths, err := s.findBookFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find book files: %w", err)
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no book files found in %s", s.libraryPath)
	}

	log.Printf("Found %d book files, starting metadata extraction...", len(filePaths))
	s.reportProgress(0, len(filePaths), fmt.Sprintf("Found %d files, starting parsing...", len(filePaths)))

	// Second pass: parse metadata with worker pool
	books, err := s.parseFilesParallelWithContext(ctx, filePaths)
	if err != nil {
		return nil, err
	}

	log.Printf("Parsing complete: %d books successfully parsed", len(books))
	return books, nil
}

// findBookFiles recursively finds all book files in the library directory
func (s *Scanner) findBookFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(s.libraryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Warning: error accessing %s: %v", path, err)
			return nil // Continue walking
		}

		if info.IsDir() {
			return nil
		}

		// Check if file has supported extension
		ext := strings.ToLower(filepath.Ext(path))

		// Handle .fb2.zip specially
		if strings.HasSuffix(strings.ToLower(path), ".fb2.zip") {
			ext = ".fb2.zip"
		}

		if s.supportedExts[ext] {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// parseFilesParallel parses files using a worker pool for parallel processing
func (s *Scanner) parseFilesParallel(filePaths []string) []*ScannedBook {
	books, _ := s.parseFilesParallelWithContext(context.Background(), filePaths)
	return books
}

// parseFilesParallelWithContext parses files with cancellation support
func (s *Scanner) parseFilesParallelWithContext(ctx context.Context, filePaths []string) ([]*ScannedBook, error) {
	jobs := make(chan string, len(filePaths))
	results := make(chan []*ScannedBook, len(filePaths))

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				// Check for cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				booksFromFile := s.parseFile(path)
				if len(booksFromFile) > 0 {
					select {
					case results <- booksFromFile:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Send jobs with cancellation check
	go func() {
		for _, path := range filePaths {
			select {
			case jobs <- path:
			case <-ctx.Done():
				close(jobs)
				return
			}
		}
		close(jobs)
	}()

	// Wait for workers and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and report progress
	var books []*ScannedBook
	parsed := 0
	total := len(filePaths)

	for {
		select {
		case <-ctx.Done():
			return books, ctx.Err()
		case booksFromFile, ok := <-results:
			if !ok {
				return books, nil
			}
			books = append(books, booksFromFile...)
			parsed++

			if parsed%10 == 0 || parsed == total {
				s.reportProgress(parsed, total, fmt.Sprintf("Parsed %d/%d archives, %d books found...", parsed, total, len(books)))
			}
		}
	}
}

// parseFile parses a single book file and extracts metadata
// For ZIP archives containing multiple FB2 files, returns multiple ScannedBook entries
func (s *Scanner) parseFile(path string) []*ScannedBook {
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("Warning: failed to stat %s: %v", path, err)
		return nil
	}

	relPath, err := filepath.Rel(s.libraryPath, path)
	if err != nil {
		relPath = filepath.Base(path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	format := strings.TrimPrefix(ext, ".")

	// Handle .fb2.zip
	if strings.HasSuffix(strings.ToLower(path), ".fb2.zip") {
		format = "fb2.zip"
	}

	// Handle regular .zip files as potential FB2 archives
	if format == "zip" {
		format = "fb2.zip"
	}

	// For ZIP archives, extract all FB2 files inside
	if format == "fb2.zip" {
		return s.parseZipArchive(path, relPath)
	}

	// For single files (EPUB, FB2)
	book := &ScannedBook{
		FilePath: path,
		RelPath:  relPath,
		FileName: strings.TrimSuffix(filepath.Base(path), ext),
		Format:   format,
		Size:     info.Size(),
	}

	// Parse metadata using the parser registry
	metadata, parseErr := parser.Parse(format, path)
	if parseErr != nil {
		log.Printf("Warning: failed to parse %s: %v", path, parseErr)
		book.ParseError = parseErr
		// Still return the book with basic file info
		return []*ScannedBook{book}
	}

	book.Metadata = metadata
	return []*ScannedBook{book}
}

// CreateLibraryForImport creates a library entry before scanning/importing
// This ensures the library exists even if the import is interrupted
func (imp *Importer) CreateLibraryForImport(libraryName, libraryPath string, firstAuthorOnly bool) (int64, error) {
	imp.libraryPath = libraryPath
	imp.firstAuthorOnly = firstAuthorOnly

	// Load genre codes
	if err := imp.loadGenreCodes(); err != nil {
		return 0, fmt.Errorf("failed to load genre codes: %w", err)
	}

	// Create library
	libID, err := imp.createLibrary(libraryName, libraryPath, "")
	if err != nil {
		return 0, fmt.Errorf("failed to create library: %w", err)
	}
	imp.libraryID = libID

	log.Printf("Created library %d: %s", libID, libraryName)
	return libID, nil
}

// ImportBooksToLibrary imports books to an existing library
func (imp *Importer) ImportBooksToLibrary(libraryID int64, books []*ScannedBook) error {
	return imp.ImportBooksToLibraryWithContext(context.Background(), libraryID, books)
}

// ImportBooksToLibraryWithContext imports books with cancellation support
func (imp *Importer) ImportBooksToLibraryWithContext(ctx context.Context, libraryID int64, books []*ScannedBook) error {
	imp.libraryID = libraryID

	log.Printf("Starting import of %d books to library %d...", len(books), libraryID)
	imp.reportProgress(0, len(books), "Starting import...")

	// Import books in batches
	batchSize := 100
	imported := 0
	skipped := 0

	for i := 0; i < len(books); i += batchSize {
		// Check for cancellation before each batch
		select {
		case <-ctx.Done():
			log.Printf("Import canceled after %d books imported", imported)
			return ctx.Err()
		default:
		}

		end := i + batchSize
		if end > len(books) {
			end = len(books)
		}
		batch := books[i:end]

		count, err := imp.importBookBatch(batch)
		if err != nil {
			log.Printf("Warning: batch import error: %v", err)
		}

		imported += count
		skipped += (len(batch) - count)

		imp.reportProgress(end, len(books), fmt.Sprintf("Imported %d books, skipped %d...", imported, skipped))
	}

	log.Printf("Import complete: %d books imported, %d skipped", imported, skipped)
	imp.reportProgress(len(books), len(books), fmt.Sprintf("Complete! %d books imported, %d skipped", imported, skipped))

	return nil
}

// ImportScannedBooks imports scanned books into the database
// This is a convenience method that creates the library and imports books in one call
func (imp *Importer) ImportScannedBooks(books []*ScannedBook, libraryName, libraryPath string, firstAuthorOnly bool) (int64, error) {
	// Create library first
	libID, err := imp.CreateLibraryForImport(libraryName, libraryPath, firstAuthorOnly)
	if err != nil {
		return 0, err
	}

	// Import books to the library
	if err := imp.ImportBooksToLibrary(libID, books); err != nil {
		return libID, err // Return library ID even on error so it can be found
	}

	return libID, nil
}

// Legacy method kept for backward compatibility
func (imp *Importer) importScannedBooksLegacy(books []*ScannedBook, libraryName, libraryPath string, firstAuthorOnly bool) (int64, error) {
	imp.libraryPath = libraryPath
	imp.firstAuthorOnly = firstAuthorOnly

	// Load genre codes
	if err := imp.loadGenreCodes(); err != nil {
		return 0, fmt.Errorf("failed to load genre codes: %w", err)
	}

	// Create library
	libID, err := imp.createLibrary(libraryName, libraryPath, "")
	if err != nil {
		return 0, fmt.Errorf("failed to create library: %w", err)
	}
	imp.libraryID = libID

	log.Printf("Starting import of %d books to library %d...", len(books), libID)
	imp.reportProgress(0, len(books), "Starting import...")

	// Import books in batches
	batchSize := 100
	imported := 0
	skipped := 0

	for i := 0; i < len(books); i += batchSize {
		end := i + batchSize
		if end > len(books) {
			end = len(books)
		}
		batch := books[i:end]

		count, err := imp.importBookBatch(batch)
		if err != nil {
			log.Printf("Warning: batch import error: %v", err)
		}

		imported += count
		skipped += (len(batch) - count)

		imp.reportProgress(end, len(books), fmt.Sprintf("Imported %d books, skipped %d...", imported, skipped))
	}

	log.Printf("Import complete: %d books imported, %d skipped", imported, skipped)
	imp.reportProgress(len(books), len(books), fmt.Sprintf("Complete! %d books imported, %d skipped", imported, skipped))

	return libID, nil
}

// importBookBatch imports a batch of books in a single transaction
func (imp *Importer) importBookBatch(books []*ScannedBook) (int, error) {
	sqlxTx, err := imp.db.Beginx()
	if err != nil {
		return 0, err
	}
	tx := &db.Tx{Tx: sqlxTx}
	defer tx.Rollback()

	count := 0
	for _, scannedBook := range books {
		if scannedBook.Metadata == nil {
			continue // Skip books with parse errors
		}

		// Convert scanned book to database book
		book := &db.Book{
			LibraryID: imp.libraryID,
			Title:     scannedBook.Metadata.Title,
			Lang:      scannedBook.Metadata.Language,
			File:      scannedBook.FileName,
			Archive:   scannedBook.Archive,
			Format:    scannedBook.Format,
			Size:      scannedBook.Size,
			Rating:    0,
			Deleted:   false,
			AddedAt:   time.Now(),
			IDInLib:   0,
			Keywords:  "",
		}

		// Use relative path as file identifier if no archive
		if book.Archive == "" {
			book.File = strings.TrimSuffix(scannedBook.RelPath, filepath.Ext(scannedBook.RelPath))
		}

		bookID, err := imp.insertBook(tx, book)
		if err != nil {
			log.Printf("Warning: failed to insert book %s: %v", book.Title, err)
			continue
		}

		// Insert authors
		var dbAuthors []db.Author
		for _, authorName := range scannedBook.Metadata.Authors {
			// Parse author name - for now, treat the whole string as LastName
			// More sophisticated parsing could split "FirstName LastName" format
			dbAuthors = append(dbAuthors, db.Author{
				LibraryID: imp.libraryID,
				LastName:  authorName,
			})
		}
		if err := imp.insertAuthors(tx, bookID, dbAuthors); err != nil {
			log.Printf("Warning: failed to insert authors for %s: %v", book.Title, err)
		}

		// Insert series
		if scannedBook.Metadata.Series != "" {
			if err := imp.insertSeries(tx, bookID, scannedBook.Metadata.Series, scannedBook.Metadata.SeriesIndex); err != nil {
				log.Printf("Warning: failed to insert series for %s: %v", book.Title, err)
			}
		}

		// Insert genres (map genre names to codes if possible)
		var genreCodes []string
		for _, genre := range scannedBook.Metadata.Genres {
			// Try to find matching genre code
			genreCode := strings.ToLower(strings.ReplaceAll(genre, " ", "_"))
			genreCodes = append(genreCodes, genreCode)
		}
		if err := imp.insertGenres(tx, bookID, genreCodes); err != nil {
			log.Printf("Warning: failed to insert genres for %s: %v", book.Title, err)
		}

		count++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Scanner) reportProgress(current, total int, message string) {
	if s.progress != nil {
		s.progress(current, total, message)
	}
}
