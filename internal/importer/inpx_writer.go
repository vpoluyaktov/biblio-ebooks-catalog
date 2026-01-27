package importer

import (
	"archive/zip"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"biblio-opds-server/internal/db"
)

// INPXWriter handles exporting library data to INPX format
type INPXWriter struct {
	db       *db.DB
	progress ProgressCallback
}

// NewINPXWriter creates a new INPX writer
func NewINPXWriter(database *db.DB) *INPXWriter {
	return &INPXWriter{
		db: database,
	}
}

// SetProgressCallback sets the progress callback function
func (w *INPXWriter) SetProgressCallback(cb ProgressCallback) {
	w.progress = cb
}

// ExportLibraryToINPX exports a library to INPX format
func (w *INPXWriter) ExportLibraryToINPX(libraryID int64, outputPath string) error {
	// Get library info
	library, err := w.db.GetLibrary(libraryID)
	if err != nil {
		return fmt.Errorf("failed to get library: %w", err)
	}

	log.Printf("Exporting library %d (%s) to %s", libraryID, library.Name, outputPath)
	w.reportProgress(0, 100, "Starting export...")

	// Get all books for this library (fetch in batches)
	var allBooks []db.Book
	batchSize := 1000
	offset := 0

	for {
		books, err := w.db.GetBooks(libraryID, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to get books: %w", err)
		}
		if len(books) == 0 {
			break
		}
		allBooks = append(allBooks, books...)
		offset += len(books)
		if len(books) < batchSize {
			break
		}
	}

	if len(allBooks) == 0 {
		return fmt.Errorf("no books found in library %d", libraryID)
	}

	log.Printf("Found %d books to export", len(allBooks))
	w.reportProgress(10, 100, fmt.Sprintf("Found %d books, building index...", len(allBooks)))

	// Build INPX records
	records, err := w.buildINPXRecords(allBooks)
	if err != nil {
		return fmt.Errorf("failed to build INPX records: %w", err)
	}

	w.reportProgress(60, 100, "Writing INPX file...")

	// Write INPX file
	if err := w.writeINPXFile(outputPath, library.Name, records); err != nil {
		return fmt.Errorf("failed to write INPX file: %w", err)
	}

	log.Printf("Successfully exported %d books to %s", len(allBooks), outputPath)
	w.reportProgress(100, 100, fmt.Sprintf("Complete! Exported %d books", len(allBooks)))

	return nil
}

// ExportLibraryByNameToINPX exports a library by name to INPX format
func (w *INPXWriter) ExportLibraryByNameToINPX(libraryName string, outputPath string) error {
	// Find library by name
	libraries, err := w.db.GetLibraries()
	if err != nil {
		return fmt.Errorf("failed to get libraries: %w", err)
	}

	var libraryID int64
	for _, lib := range libraries {
		if lib.Name == libraryName {
			libraryID = lib.ID
			break
		}
	}

	if libraryID == 0 {
		return fmt.Errorf("library not found: %s", libraryName)
	}

	return w.ExportLibraryToINPX(libraryID, outputPath)
}

// buildINPXRecords builds INPX records from database books
func (w *INPXWriter) buildINPXRecords(books []db.Book) ([]string, error) {
	records := make([]string, 0, len(books))
	total := len(books)

	for i, book := range books {
		if i%100 == 0 {
			progress := 10 + (i * 50 / total) // 10-60% range
			w.reportProgress(progress, 100, fmt.Sprintf("Processing book %d/%d...", i+1, total))
		}

		record, err := w.buildINPXRecord(&book)
		if err != nil {
			log.Printf("Warning: failed to build record for book %d: %v", book.ID, err)
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// buildINPXRecord builds a single INPX record from a book
func (w *INPXWriter) buildINPXRecord(book *db.Book) (string, error) {
	// Get all authors for this book
	authors, err := w.db.GetBookAuthors(book.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get authors: %w", err)
	}

	// Format authors as "LastName,FirstName,MiddleName:LastName2,FirstName2,MiddleName2"
	var authorStrs []string
	for _, author := range authors {
		parts := []string{author.LastName}
		if author.FirstName != "" {
			parts = append(parts, author.FirstName)
		}
		if author.MiddleName != "" {
			parts = append(parts, author.MiddleName)
		}
		authorStrs = append(authorStrs, strings.Join(parts, ","))
	}
	authorsStr := strings.Join(authorStrs, ":")

	// Get genres for this book
	genres, err := w.db.GetBookGenres(book.ID)
	if err != nil {
		log.Printf("Warning: failed to get genres for book %d: %v", book.ID, err)
		genres = []db.Genre{} // Continue without genres
	}

	// Format genres as "code1:code2:code3"
	var genreCodes []string
	for _, genre := range genres {
		if genre.Code != "" {
			// Take first code if multiple codes exist
			codes := strings.Split(genre.Code, ",")
			if len(codes) > 0 {
				genreCodes = append(genreCodes, strings.TrimSpace(codes[0]))
			}
		}
	}
	genresStr := strings.Join(genreCodes, ":")

	// Get series info
	var seriesName string
	var seqNum int
	seriesList, err := w.db.GetBookSeries(book.ID)
	if err == nil && len(seriesList) > 0 {
		// Use first series if multiple exist
		seriesName = seriesList[0].Name
		// Get sequence number from book_series table
		var bs db.BookSeries
		err = w.db.Get(&bs, "SELECT seq_num FROM book_series WHERE book_id = ? AND series_id = ?",
			book.ID, seriesList[0].ID)
		if err == nil {
			seqNum = bs.SeqNum
		}
	}

	// Format date
	dateStr := book.AddedAt.Format("2006-01-02")

	// Build INPX record with field separator (0x04)
	// Fields: Author;Genre;Title;Series;SeriesNum;File;Size;LibId;Deleted;Ext;Date;Lang;Rating;Keywords
	fields := []string{
		authorsStr,                          // 0: Authors
		genresStr,                           // 1: Genres
		book.Title,                          // 2: Title
		seriesName,                          // 3: Series
		strconv.Itoa(seqNum),                // 4: Series number
		book.File,                           // 5: File
		strconv.FormatInt(book.Size, 10),    // 6: Size
		strconv.FormatInt(book.IDInLib, 10), // 7: LibID
		"0",                                 // 8: Deleted (0 = not deleted)
		book.Format,                         // 9: Format/Extension
		dateStr,                             // 10: Date
		book.Lang,                           // 11: Language
		strconv.Itoa(book.Rating),           // 12: Rating
		book.Keywords,                       // 13: Keywords
	}

	return strings.Join(fields, fieldSeparator), nil
}

// writeINPXFile writes records to an INPX file
func (w *INPXWriter) writeINPXFile(outputPath, libraryName string, records []string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create ZIP file
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create INPX file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Create .inp file inside ZIP
	inpFileName := strings.ReplaceAll(libraryName, " ", "_") + ".inp"
	inpWriter, err := zipWriter.Create(inpFileName)
	if err != nil {
		return fmt.Errorf("failed to create .inp file in archive: %w", err)
	}

	// Write records
	for _, record := range records {
		if _, err := inpWriter.Write([]byte(record + "\n")); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	// Create structure.info file
	structWriter, err := zipWriter.Create("structure.info")
	if err != nil {
		return fmt.Errorf("failed to create structure.info: %w", err)
	}

	// Write structure info with field names
	structureInfo := "AUTHOR;GENRE;TITLE;SERIES;SERNO;FILE;SIZE;LIBID;DEL;EXT;DATE;LANG;STARS;KEYWORDS\n"
	if _, err := structWriter.Write([]byte(structureInfo)); err != nil {
		return fmt.Errorf("failed to write structure.info: %w", err)
	}

	// Create collection.info file with metadata
	collWriter, err := zipWriter.Create("collection.info")
	if err != nil {
		return fmt.Errorf("failed to create collection.info: %w", err)
	}

	collectionInfo := fmt.Sprintf("name=%s\nfile=%s\nversion=1.0\ndate=%s\n",
		libraryName,
		inpFileName,
		time.Now().Format("2006-01-02"),
	)
	if _, err := collWriter.Write([]byte(collectionInfo)); err != nil {
		return fmt.Errorf("failed to write collection.info: %w", err)
	}

	return nil
}

func (w *INPXWriter) reportProgress(current, total int, message string) {
	if w.progress != nil {
		w.progress(current, total, message)
	}
}
