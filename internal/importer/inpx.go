package importer

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"biblio-catalog/internal/db"
)

const fieldSeparator = "\x04"

type fieldIndex struct {
	Authors  int
	Genres   int
	Title    int
	Series   int
	SeqNum   int
	File     int
	Size     int
	LibID    int
	Deleted  int
	Format   int
	Date     int
	Lang     int
	Rating   int
	Keywords int
	Folder   int
}

func defaultFieldIndex() fieldIndex {
	return fieldIndex{
		Authors:  0,
		Genres:   1,
		Title:    2,
		Series:   3,
		SeqNum:   4,
		File:     5,
		Size:     6,
		LibID:    7,
		Deleted:  8,
		Format:   9,
		Date:     10,
		Lang:     11,
		Rating:   12,
		Keywords: 13,
		Folder:   -1,
	}
}

type ProgressCallback func(current, total int, message string)

// ZipProgressCallback provides detailed progress for ZIP file processing
type ZipProgressCallback func(fileIndex, fileTotal int, zipCurrent, zipTotal int, zipFileName, message string)

type Importer struct {
	db              *db.DB
	libraryID       int64
	libraryPath     string
	firstAuthorOnly bool
	progress        ProgressCallback

	authors    map[string]int64 // "last|first|middle" -> id
	series     map[string]int64 // name -> id
	genreCodes map[string]int   // code -> genre_id
}

func New(database *db.DB) *Importer {
	return &Importer{
		db:         database,
		authors:    make(map[string]int64),
		series:     make(map[string]int64),
		genreCodes: make(map[string]int),
	}
}

func (imp *Importer) SetProgressCallback(cb ProgressCallback) {
	imp.progress = cb
}

func (imp *Importer) reportProgress(current, total int, message string) {
	if imp.progress != nil {
		imp.progress(current, total, message)
	}
}

func (imp *Importer) ImportINPX(inpxPath, libraryName, libraryPath string, firstAuthorOnly bool) (int64, error) {
	imp.libraryPath = libraryPath
	imp.firstAuthorOnly = firstAuthorOnly

	// Load genre codes
	if err := imp.loadGenreCodes(); err != nil {
		return 0, fmt.Errorf("failed to load genre codes: %w", err)
	}

	// Create or get library
	libID, err := imp.createLibrary(libraryName, libraryPath, inpxPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create library: %w", err)
	}
	imp.libraryID = libID

	// Open INPX file
	r, err := zip.OpenReader(inpxPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open INPX: %w", err)
	}
	defer r.Close()

	// Parse structure.info if exists
	fields := defaultFieldIndex()
	for _, f := range r.File {
		if strings.ToLower(f.Name) == "structure.info" {
			fields, err = imp.parseStructureInfo(f)
			if err != nil {
				log.Printf("Warning: failed to parse structure.info: %v", err)
				fields = defaultFieldIndex()
			}
			break
		}
	}

	// Count .inp files for progress
	var inpFiles []*zip.File
	for _, f := range r.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".inp") {
			inpFiles = append(inpFiles, f)
		}
	}

	imp.reportProgress(0, len(inpFiles), "0 books imported. Starting...")

	// Process all .inp files
	var totalBooks int
	for i, f := range inpFiles {
		count, err := imp.processINPFile(f, fields)
		if err != nil {
			log.Printf("Warning: failed to process %s: %v", f.Name, err)
			continue
		}
		totalBooks += count
		log.Printf("Processed %s: %d books", f.Name, count)

		// Report total book count and current file being processed
		imp.reportProgress(i+1, len(inpFiles), fmt.Sprintf("%d books imported. Processing %s...", totalBooks, f.Name))
	}

	imp.reportProgress(len(inpFiles), len(inpFiles), fmt.Sprintf("%d books imported. Complete!", totalBooks))
	log.Printf("Import complete: %d books total", totalBooks)
	return libID, nil
}

func (imp *Importer) loadGenreCodes() error {
	genres, err := imp.db.GetGenres()
	if err != nil {
		return err
	}

	for _, g := range genres {
		if g.Code == "" {
			continue
		}
		codes := strings.Split(g.Code, ",")
		for _, code := range codes {
			code = strings.TrimSpace(code)
			if code != "" {
				imp.genreCodes[code] = g.ID
			}
		}
	}
	return nil
}

func (imp *Importer) createLibrary(name, path, inpxPath string) (int64, error) {
	lib := &db.Library{
		Name:     name,
		Path:     path,
		InpxPath: inpxPath,
	}
	return imp.db.CreateLibrary(lib)
}

func (imp *Importer) parseStructureInfo(f *zip.File) (fieldIndex, error) {
	rc, err := f.Open()
	if err != nil {
		return defaultFieldIndex(), err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return defaultFieldIndex(), err
	}

	fields := fieldIndex{
		Authors:  -1,
		Genres:   -1,
		Title:    -1,
		Series:   -1,
		SeqNum:   -1,
		File:     -1,
		Size:     -1,
		LibID:    -1,
		Deleted:  -1,
		Format:   -1,
		Date:     -1,
		Lang:     -1,
		Rating:   -1,
		Keywords: -1,
		Folder:   -1,
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(strings.ToUpper(line), ";")
		for i, part := range parts {
			switch strings.TrimSpace(part) {
			case "AUTHOR":
				fields.Authors = i
			case "GENRE":
				fields.Genres = i
			case "TITLE":
				fields.Title = i
			case "SERIES":
				fields.Series = i
			case "SERNO":
				fields.SeqNum = i
			case "FILE":
				fields.File = i
			case "SIZE":
				fields.Size = i
			case "LIBID":
				fields.LibID = i
			case "DEL":
				fields.Deleted = i
			case "EXT":
				fields.Format = i
			case "DATE":
				fields.Date = i
			case "LANG":
				fields.Lang = i
			case "STARS":
				fields.Rating = i
			case "KEYWORDS":
				fields.Keywords = i
			case "FOLDER":
				fields.Folder = i
			}
		}
	}

	return fields, nil
}

func (imp *Importer) processINPFile(f *zip.File, fields fieldIndex) (int, error) {
	rc, err := f.Open()
	if err != nil {
		return 0, err
	}
	defer rc.Close()

	archiveName := strings.TrimSuffix(f.Name, filepath.Ext(f.Name)) + ".zip"

	sqlxTx, err := imp.db.Beginx()
	if err != nil {
		return 0, err
	}
	tx := &db.Tx{Tx: sqlxTx}
	defer tx.Rollback()

	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, fieldSeparator)

		book, authors, seriesName, seqNum, genreCodes, err := imp.parseLine(parts, fields, archiveName)
		if err != nil {
			continue
		}

		bookID, err := imp.insertBook(tx, book)
		if err != nil {
			log.Printf("Warning: failed to insert book %s: %v", book.Title, err)
			continue
		}

		if err := imp.insertAuthors(tx, bookID, authors); err != nil {
			log.Printf("Warning: failed to insert authors for %s: %v", book.Title, err)
		}

		if seriesName != "" {
			if err := imp.insertSeries(tx, bookID, seriesName, seqNum); err != nil {
				log.Printf("Warning: failed to insert series for %s: %v", book.Title, err)
			}
		}

		if err := imp.insertGenres(tx, bookID, genreCodes); err != nil {
			log.Printf("Warning: failed to insert genres for %s: %v", book.Title, err)
		}

		count++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return count, scanner.Err()
}

func (imp *Importer) parseLine(parts []string, fields fieldIndex, archiveName string) (*db.Book, []db.Author, string, int, []string, error) {
	getField := func(idx int) string {
		if idx >= 0 && idx < len(parts) {
			return strings.TrimSpace(parts[idx])
		}
		return ""
	}

	title := getField(fields.Title)
	if title == "" {
		return nil, nil, "", 0, nil, fmt.Errorf("empty title")
	}

	file := getField(fields.File)
	format := strings.ToLower(getField(fields.Format))
	size, _ := strconv.ParseInt(getField(fields.Size), 10, 64)
	idInLib, _ := strconv.ParseInt(getField(fields.LibID), 10, 64)
	// In Flibusta INP format, field 8 is NOT deleted flag - it's a type/version field
	// Deleted books are typically in separate "_lost" archives or have empty format
	deleted := false
	lang := getField(fields.Lang)
	if len(lang) > 2 {
		lang = lang[:2]
	}
	rating, _ := strconv.Atoi(getField(fields.Rating))
	keywords := getField(fields.Keywords)

	dateStr := getField(fields.Date)
	var addedAt time.Time
	if dateStr != "" {
		addedAt, _ = time.Parse("2006-01-02", dateStr)
	}
	if addedAt.IsZero() {
		addedAt = time.Now()
	}

	book := &db.Book{
		LibraryID: imp.libraryID,
		Title:     title,
		Lang:      lang,
		File:      file,
		Archive:   archiveName,
		Format:    format,
		Size:      size,
		Rating:    rating,
		Deleted:   deleted,
		AddedAt:   addedAt,
		IDInLib:   idInLib,
		Keywords:  keywords,
	}

	// Parse authors
	authorsStr := getField(fields.Authors)
	var authors []db.Author
	if authorsStr != "" {
		authorParts := strings.Split(authorsStr, ":")
		for _, ap := range authorParts {
			ap = strings.TrimSpace(ap)
			if ap == "" {
				continue
			}
			nameParts := strings.Split(ap, ",")
			author := db.Author{LibraryID: imp.libraryID}
			if len(nameParts) > 0 {
				author.LastName = strings.TrimSpace(nameParts[0])
			}
			if len(nameParts) > 1 {
				author.FirstName = strings.TrimSpace(nameParts[1])
			}
			if len(nameParts) > 2 {
				author.MiddleName = strings.TrimSpace(nameParts[2])
			}
			if author.LastName != "" || author.FirstName != "" {
				authors = append(authors, author)
			}
		}
	}

	// Parse series
	seriesName := getField(fields.Series)
	seqNum, _ := strconv.Atoi(getField(fields.SeqNum))

	// Parse genres
	genresStr := getField(fields.Genres)
	var genreCodes []string
	if genresStr != "" {
		genreCodes = strings.Split(genresStr, ":")
	}

	return book, authors, seriesName, seqNum, genreCodes, nil
}

func (imp *Importer) insertBook(tx *db.Tx, book *db.Book) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO book (library_id, title, lang, file, archive, format, size, rating, deleted, added_at, id_in_lib, first_author_id, keywords)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?)`,
		book.LibraryID, book.Title, book.Lang, book.File, book.Archive, book.Format,
		book.Size, book.Rating, book.Deleted, book.AddedAt, book.IDInLib, book.Keywords,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (imp *Importer) insertAuthors(tx *db.Tx, bookID int64, authors []db.Author) error {
	for i, author := range authors {
		if imp.firstAuthorOnly && i > 0 {
			break
		}

		key := fmt.Sprintf("%s|%s|%s", author.LastName, author.FirstName, author.MiddleName)
		authorID, exists := imp.authors[key]

		if !exists {
			result, err := tx.Exec(`
				INSERT INTO author (library_id, last_name, first_name, middle_name)
				VALUES (?, ?, ?, ?)`,
				author.LibraryID, author.LastName, author.FirstName, author.MiddleName,
			)
			if err != nil {
				return err
			}
			authorID, _ = result.LastInsertId()
			imp.authors[key] = authorID
		}

		_, err := tx.Exec(`INSERT OR IGNORE INTO book_author (book_id, author_id) VALUES (?, ?)`,
			bookID, authorID)
		if err != nil {
			return err
		}

		if i == 0 {
			_, err = tx.Exec(`UPDATE book SET first_author_id = ? WHERE id = ?`, authorID, bookID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (imp *Importer) insertSeries(tx *db.Tx, bookID int64, seriesName string, seqNum int) error {
	seriesID, exists := imp.series[seriesName]

	if !exists {
		result, err := tx.Exec(`INSERT INTO series (library_id, name) VALUES (?, ?)`,
			imp.libraryID, seriesName)
		if err != nil {
			return err
		}
		seriesID, _ = result.LastInsertId()
		imp.series[seriesName] = seriesID
	}

	_, err := tx.Exec(`INSERT OR IGNORE INTO book_series (book_id, series_id, seq_num) VALUES (?, ?, ?)`,
		bookID, seriesID, seqNum)
	return err
}

func (imp *Importer) insertGenres(tx *db.Tx, bookID int64, codes []string) error {
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}

		genreID, exists := imp.genreCodes[code]
		if !exists {
			continue
		}

		_, err := tx.Exec(`INSERT OR IGNORE INTO book_genre (book_id, genre_id) VALUES (?, ?)`,
			bookID, genreID)
		if err != nil {
			return err
		}
	}
	return nil
}
