package importer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"biblio-opds-server/internal/bookfile"
	"biblio-opds-server/internal/db"
)

// ScannedBook represents a book found during directory scanning
type ScannedBook struct {
	FilePath   string
	RelPath    string // Relative path from library root
	FileName   string
	Format     string
	Size       int64
	Metadata   *bookfile.EPUBMetadata
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
		},
	}
}

// SetProgressCallback sets the progress callback function
func (s *Scanner) SetProgressCallback(cb ProgressCallback) {
	s.progress = cb
}

// ScanDirectory scans the library directory and returns all found books
func (s *Scanner) ScanDirectory() ([]*ScannedBook, error) {
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
	books := s.parseFilesParallel(filePaths)

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
	jobs := make(chan string, len(filePaths))
	results := make(chan *ScannedBook, len(filePaths))

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				book := s.parseFile(path)
				if book != nil {
					results <- book
				}
			}
		}()
	}

	// Send jobs
	for _, path := range filePaths {
		jobs <- path
	}
	close(jobs)

	// Wait for workers and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and report progress
	var books []*ScannedBook
	parsed := 0
	total := len(filePaths)

	for book := range results {
		books = append(books, book)
		parsed++

		if parsed%10 == 0 || parsed == total {
			s.reportProgress(parsed, total, fmt.Sprintf("Parsed %d/%d files...", parsed, total))
		}
	}

	return books
}

// parseFile parses a single book file and extracts metadata
func (s *Scanner) parseFile(path string) *ScannedBook {
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

	book := &ScannedBook{
		FilePath: path,
		RelPath:  relPath,
		FileName: strings.TrimSuffix(filepath.Base(path), ext),
		Format:   format,
		Size:     info.Size(),
	}

	// Parse metadata based on format
	var metadata *bookfile.EPUBMetadata
	var parseErr error

	switch format {
	case "epub":
		metadata, parseErr = bookfile.ParseEPUBMetadata(path)
	case "fb2":
		metadata, parseErr = bookfile.ParseFB2Metadata(path)
	case "fb2.zip":
		metadata, parseErr = bookfile.ParseFB2MetadataFromZip(path)
	default:
		parseErr = fmt.Errorf("unsupported format: %s", format)
	}

	if parseErr != nil {
		log.Printf("Warning: failed to parse %s: %v", path, parseErr)
		book.ParseError = parseErr
		// Still return the book with basic file info
		return book
	}

	book.Metadata = metadata
	return book
}

// ImportScannedBooks imports scanned books into the database
func (imp *Importer) ImportScannedBooks(books []*ScannedBook, libraryName, libraryPath string, firstAuthorOnly bool) (int64, error) {
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
		for _, author := range scannedBook.Metadata.Authors {
			dbAuthors = append(dbAuthors, db.Author{
				LibraryID:  imp.libraryID,
				LastName:   author.LastName,
				FirstName:  author.FirstName,
				MiddleName: author.MiddleName,
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
