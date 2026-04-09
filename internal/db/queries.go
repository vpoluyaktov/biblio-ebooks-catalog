package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// buildLangFilter returns a SQL WHERE clause fragment and args for language filtering.
// If langs is empty, returns empty string and no args (no filtering).
// tableAlias is the book table alias (e.g., "b" or "book").
func buildLangFilter(tableAlias string, langs []string) (string, []interface{}) {
	if len(langs) == 0 {
		return "", nil
	}
	placeholders := make([]string, len(langs))
	args := make([]interface{}, len(langs))
	for i, l := range langs {
		placeholders[i] = "?"
		args[i] = l
	}
	clause := fmt.Sprintf(" AND %s.lang IN (%s)", tableAlias, strings.Join(placeholders, ","))
	return clause, args
}

// buildLangJoinCond returns a SQL JOIN ON condition fragment for language filtering.
// Used when the lang filter is applied as part of a JOIN condition rather than a WHERE clause.
func buildLangJoinCond(tableAlias string, langs []string) string {
	if len(langs) == 0 {
		return ""
	}
	placeholders := make([]string, len(langs))
	for i := range langs {
		placeholders[i] = "?"
	}
	return fmt.Sprintf(" AND %s.lang IN (%s)", tableAlias, strings.Join(placeholders, ","))
}

func (db *DB) GetLibraries() ([]Library, error) {
	var libraries []Library
	err := db.Select(&libraries, "SELECT * FROM library ORDER BY name")
	return libraries, err
}

func (db *DB) GetLibrary(id int64) (*Library, error) {
	var lib Library
	err := db.Get(&lib, "SELECT * FROM library WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &lib, nil
}

func (db *DB) CreateLibrary(lib *Library) (int64, error) {
	// Find the lowest available ID (reuse deleted IDs)
	nextID := int64(1)

	// Check if ID 1 is available
	var count int
	db.Get(&count, "SELECT COUNT(*) FROM library WHERE id = 1")
	if count > 0 {
		// ID 1 is taken, find first gap in sequence
		err := db.Get(&nextID, `
			SELECT MIN(id) + 1 FROM library l1 
			WHERE NOT EXISTS (SELECT 1 FROM library l2 WHERE l2.id = l1.id + 1)`)
		if err != nil || nextID == 0 {
			// No gaps found, use max + 1
			db.Get(&nextID, "SELECT COALESCE(MAX(id), 0) + 1 FROM library")
		}
	}

	_, err := db.Exec(`
		INSERT INTO library (id, name, path, inpx, version, first_author, without_deleted, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		nextID, lib.Name, lib.Path, lib.InpxPath, lib.Version, lib.FirstAuthorOnly, lib.WithoutDeleted, true,
	)
	if err != nil {
		return 0, err
	}
	return nextID, nil
}

func (db *DB) UpdateLibrary(lib *Library) error {
	_, err := db.Exec(`
		UPDATE library SET name = ?, path = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		lib.Name, lib.Path, lib.Enabled, lib.ID,
	)
	return err
}

func (db *DB) SetLibraryEnabled(id int64, enabled bool) error {
	_, err := db.Exec(`
		UPDATE library SET enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		enabled, id,
	)
	return err
}

func (db *DB) GetLibraryStats(id int64) (bookCount int64, authorCount int64, seriesCount int64, err error) {
	err = db.Get(&bookCount, "SELECT COUNT(*) FROM book WHERE library_id = ? AND deleted = 0", id)
	if err != nil {
		return
	}
	err = db.Get(&authorCount, "SELECT COUNT(*) FROM author WHERE library_id = ?", id)
	if err != nil {
		return
	}
	err = db.Get(&seriesCount, "SELECT COUNT(*) FROM series WHERE library_id = ?", id)
	return
}

func (db *DB) GetEnabledLibraries() ([]Library, error) {
	var libraries []Library
	err := db.Select(&libraries, "SELECT * FROM library WHERE enabled = 1 ORDER BY name")
	return libraries, err
}

func (db *DB) DeleteLibrary(id int64) error {
	_, err := db.Exec("DELETE FROM library WHERE id = ?", id)
	return err
}

func (db *DB) GetGenres() ([]Genre, error) {
	var genres []Genre
	err := db.Select(&genres, "SELECT * FROM genre ORDER BY parent_id, id")
	return genres, err
}

func (db *DB) GetTopLevelGenres() ([]Genre, error) {
	var genres []Genre
	err := db.Select(&genres, "SELECT * FROM genre WHERE parent_id = 0 ORDER BY id")
	return genres, err
}

func (db *DB) GetSubGenres(parentID int) ([]Genre, error) {
	var genres []Genre
	err := db.Select(&genres, "SELECT * FROM genre WHERE parent_id = ? ORDER BY id", parentID)
	return genres, err
}

func (db *DB) GetGenreByCode(code string) (*Genre, error) {
	var genre Genre
	err := db.Get(&genre, "SELECT * FROM genre WHERE code LIKE ? LIMIT 1", "%"+code+"%")
	if err != nil {
		return nil, err
	}
	return &genre, nil
}

func (db *DB) GetAuthors(libraryID int64, limit, offset int, langs []string) ([]AuthorWithCount, error) {
	var authors []AuthorWithCount
	if len(langs) == 0 {
		err := db.Select(&authors, `
			SELECT a.*, COUNT(ba.book_id) as book_count
			FROM author a
			LEFT JOIN book_author ba ON a.id = ba.author_id
			WHERE a.library_id = ?
			GROUP BY a.id
			ORDER BY a.last_name, a.first_name
			LIMIT ? OFFSET ?`,
			libraryID, limit, offset,
		)
		return authors, err
	}
	_, langArgs := buildLangFilter("b", langs)
	langJoin := buildLangJoinCond("b", langs)
	query := `
		SELECT a.*, COUNT(DISTINCT b.id) as book_count
		FROM author a
		JOIN book_author ba ON a.id = ba.author_id
		JOIN book b ON b.id = ba.book_id AND b.deleted = 0` + langJoin + `
		WHERE a.library_id = ?
		GROUP BY a.id
		ORDER BY a.last_name, a.first_name
		LIMIT ? OFFSET ?`
	args := append(langArgs, libraryID, limit, offset)
	err := db.Select(&authors, query, args...)
	return authors, err
}

type AuthorsResult struct {
	Authors []AuthorWithCount `json:"authors"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
	HasMore bool              `json:"has_more"`
}

func (db *DB) GetAuthorsFiltered(libraryID int64, filter string, limit, offset int, langs []string) (*AuthorsResult, error) {
	var total int
	var authors []AuthorWithCount

	// Build filter condition
	filterCond := ""
	filterArgs := []interface{}{}
	if filter != "" {
		filterCond = " AND (a.last_name LIKE ? OR a.first_name LIKE ? OR a.middle_name LIKE ?)"
		filterPattern := "%" + filter + "%"
		filterArgs = append(filterArgs, filterPattern, filterPattern, filterPattern)
	}

	langClause, langArgs := buildLangFilter("b", langs)

	if len(langs) == 0 {
		// No language filter: use existing LEFT JOIN approach
		countArgs := append([]interface{}{libraryID}, filterArgs...)
		countQuery := `SELECT COUNT(DISTINCT a.id) FROM author a WHERE a.library_id = ?` + filterCond
		if err := db.Get(&total, countQuery, countArgs...); err != nil {
			return nil, err
		}

		dataArgs := append([]interface{}{libraryID}, filterArgs...)
		dataArgs = append(dataArgs, limit, offset)
		query := `
			SELECT a.*, COUNT(ba.book_id) as book_count
			FROM author a
			LEFT JOIN book_author ba ON a.id = ba.author_id
			WHERE a.library_id = ?` + filterCond + `
			GROUP BY a.id
			ORDER BY a.last_name, a.first_name
			LIMIT ? OFFSET ?`
		if err := db.Select(&authors, query, dataArgs...); err != nil {
			return nil, err
		}
	} else {
		// Language filter: use EXISTS subquery for count, JOIN for data
		langJoin := buildLangJoinCond("b", langs)

		countArgs := append([]interface{}{libraryID}, filterArgs...)
		countArgs = append(countArgs, langArgs...)
		countQuery := `SELECT COUNT(DISTINCT a.id) FROM author a
			WHERE a.library_id = ?` + filterCond + `
			AND EXISTS (
				SELECT 1 FROM book_author ba
				JOIN book b ON b.id = ba.book_id
				WHERE ba.author_id = a.id AND b.deleted = 0` + langClause + `
			)`
		if err := db.Get(&total, countQuery, countArgs...); err != nil {
			return nil, err
		}

		dataArgs := append(langArgs, libraryID)
		dataArgs = append(dataArgs, filterArgs...)
		dataArgs = append(dataArgs, limit, offset)
		query := `
			SELECT a.*, COUNT(DISTINCT b.id) as book_count
			FROM author a
			JOIN book_author ba ON a.id = ba.author_id
			JOIN book b ON b.id = ba.book_id AND b.deleted = 0` + langJoin + `
			WHERE a.library_id = ?` + filterCond + `
			GROUP BY a.id
			ORDER BY a.last_name, a.first_name
			LIMIT ? OFFSET ?`
		if err := db.Select(&authors, query, dataArgs...); err != nil {
			return nil, err
		}
	}

	return &AuthorsResult{
		Authors: authors,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+len(authors) < total,
	}, nil
}

func (db *DB) GetAuthorsByLetter(libraryID int64, letter string, langs []string) ([]AuthorWithCount, error) {
	var authors []AuthorWithCount
	if len(langs) == 0 {
		err := db.Select(&authors, `
			SELECT a.*, COUNT(ba.book_id) as book_count
			FROM author a
			LEFT JOIN book_author ba ON a.id = ba.author_id
			WHERE a.library_id = ? AND a.last_name LIKE ?
			GROUP BY a.id
			ORDER BY a.last_name, a.first_name`,
			libraryID, letter+"%",
		)
		return authors, err
	}
	_, langArgs := buildLangFilter("b", langs)
	langJoin := buildLangJoinCond("b", langs)
	args := append(langArgs, libraryID, letter+"%")
	err := db.Select(&authors, `
		SELECT a.*, COUNT(DISTINCT b.id) as book_count
		FROM author a
		JOIN book_author ba ON a.id = ba.author_id
		JOIN book b ON b.id = ba.book_id AND b.deleted = 0`+langJoin+`
		WHERE a.library_id = ? AND a.last_name LIKE ?
		GROUP BY a.id
		ORDER BY a.last_name, a.first_name`,
		args...,
	)
	return authors, err
}

// CountAuthorsByPrefix counts authors whose last_name starts with the given prefix
func (db *DB) CountAuthorsByPrefix(libraryID int64, prefix string, langs []string) (int, error) {
	var count int
	if len(langs) == 0 {
		err := db.Get(&count, `
			SELECT COUNT(DISTINCT a.id) FROM author a
			WHERE a.library_id = ? AND a.last_name LIKE ?`,
			libraryID, prefix+"%",
		)
		return count, err
	}
	langClause, langArgs := buildLangFilter("b", langs)
	args := append([]interface{}{libraryID, prefix + "%"}, langArgs...)
	err := db.Get(&count, `
		SELECT COUNT(DISTINCT a.id) FROM author a
		WHERE a.library_id = ? AND a.last_name LIKE ?
		AND EXISTS (
			SELECT 1 FROM book_author ba
			JOIN book b ON b.id = ba.book_id
			WHERE ba.author_id = a.id AND b.deleted = 0`+langClause+`
		)`,
		args...,
	)
	return count, err
}

// GetAuthorPrefixCounts returns counts for each possible next character after the given prefix
// This is used for adaptive navigation to determine if we need to drill down further
func (db *DB) GetAuthorPrefixCounts(libraryID int64, prefix string, alphabet string, langs []string) (map[string]int, error) {
	counts := make(map[string]int)

	for _, char := range alphabet {
		nextPrefix := prefix + string(char)
		count, err := db.CountAuthorsByPrefix(libraryID, nextPrefix, langs)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			counts[string(char)] = count
		}
	}

	return counts, nil
}

func (db *DB) GetAuthor(id int64) (*Author, error) {
	var author Author
	err := db.Get(&author, "SELECT * FROM author WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &author, nil
}

func (db *DB) GetSeries(libraryID int64, limit, offset int, langs []string) ([]SeriesWithCount, int64, error) {
	var total int64
	if len(langs) == 0 {
		err := db.Get(&total, "SELECT COUNT(*) FROM series WHERE library_id = ?", libraryID)
		if err != nil {
			return nil, 0, err
		}
		var series []SeriesWithCount
		err = db.Select(&series, `
			SELECT s.*, COUNT(bs.book_id) as book_count
			FROM series s
			LEFT JOIN book_series bs ON s.id = bs.series_id
			WHERE s.library_id = ?
			GROUP BY s.id
			ORDER BY s.name
			LIMIT ? OFFSET ?`,
			libraryID, limit, offset,
		)
		return series, total, err
	}

	langClause, langArgs := buildLangFilter("b", langs)
	langJoin := buildLangJoinCond("b", langs)

	countArgs := append([]interface{}{libraryID}, langArgs...)
	if err := db.Get(&total, `
		SELECT COUNT(DISTINCT s.id) FROM series s
		WHERE s.library_id = ?
		AND EXISTS (
			SELECT 1 FROM book_series bs
			JOIN book b ON b.id = bs.book_id
			WHERE bs.series_id = s.id AND b.deleted = 0`+langClause+`
		)`, countArgs...); err != nil {
		return nil, 0, err
	}

	var series []SeriesWithCount
	dataArgs := append(langArgs, libraryID, limit, offset)
	err := db.Select(&series, `
		SELECT s.*, COUNT(DISTINCT b.id) as book_count
		FROM series s
		JOIN book_series bs ON s.id = bs.series_id
		JOIN book b ON b.id = bs.book_id AND b.deleted = 0`+langJoin+`
		WHERE s.library_id = ?
		GROUP BY s.id
		ORDER BY s.name
		LIMIT ? OFFSET ?`,
		dataArgs...,
	)
	return series, total, err
}

type SeriesResult struct {
	Series  []SeriesWithCount `json:"series"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
	HasMore bool              `json:"has_more"`
}

func (db *DB) GetSeriesFiltered(libraryID int64, filter string, limit, offset int, langs []string) (*SeriesResult, error) {
	var total int
	var series []SeriesWithCount

	// Build filter condition
	filterCond := ""
	filterArgs := []interface{}{}
	if filter != "" {
		filterCond = " AND s.name LIKE ?"
		filterArgs = append(filterArgs, "%"+filter+"%")
	}

	langClause, langArgs := buildLangFilter("b", langs)

	if len(langs) == 0 {
		// No language filter: use existing LEFT JOIN approach
		countArgs := append([]interface{}{libraryID}, filterArgs...)
		if err := db.Get(&total, `SELECT COUNT(*) FROM series s WHERE s.library_id = ?`+filterCond, countArgs...); err != nil {
			return nil, err
		}
		dataArgs := append([]interface{}{libraryID}, filterArgs...)
		dataArgs = append(dataArgs, limit, offset)
		if err := db.Select(&series, `
			SELECT s.*, COUNT(bs.book_id) as book_count
			FROM series s
			LEFT JOIN book_series bs ON s.id = bs.series_id
			WHERE s.library_id = ?`+filterCond+`
			GROUP BY s.id
			ORDER BY s.name
			LIMIT ? OFFSET ?`, dataArgs...); err != nil {
			return nil, err
		}
	} else {
		// Language filter: use EXISTS subquery for count, JOIN for data
		langJoin := buildLangJoinCond("b", langs)

		countArgs := append([]interface{}{libraryID}, filterArgs...)
		countArgs = append(countArgs, langArgs...)
		if err := db.Get(&total, `
			SELECT COUNT(DISTINCT s.id) FROM series s
			WHERE s.library_id = ?`+filterCond+`
			AND EXISTS (
				SELECT 1 FROM book_series bs
				JOIN book b ON b.id = bs.book_id
				WHERE bs.series_id = s.id AND b.deleted = 0`+langClause+`
			)`, countArgs...); err != nil {
			return nil, err
		}

		dataArgs := append(langArgs, libraryID)
		dataArgs = append(dataArgs, filterArgs...)
		dataArgs = append(dataArgs, limit, offset)
		if err := db.Select(&series, `
			SELECT s.*, COUNT(DISTINCT b.id) as book_count
			FROM series s
			JOIN book_series bs ON s.id = bs.series_id
			JOIN book b ON b.id = bs.book_id AND b.deleted = 0`+langJoin+`
			WHERE s.library_id = ?`+filterCond+`
			GROUP BY s.id
			ORDER BY s.name
			LIMIT ? OFFSET ?`, dataArgs...); err != nil {
			return nil, err
		}
	}

	return &SeriesResult{
		Series:  series,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+len(series) < total,
	}, nil
}

func (db *DB) GetSeriesByID(id int64) (*Series, error) {
	var series Series
	err := db.Get(&series, "SELECT * FROM series WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &series, nil
}

func (db *DB) GetBooks(libraryID int64, limit, offset int, langs []string) ([]Book, error) {
	var books []Book
	langClause, langArgs := buildLangFilter("book", langs)
	args := append([]interface{}{libraryID}, langArgs...)
	args = append(args, limit, offset)
	err := db.Select(&books, `
		SELECT * FROM book
		WHERE library_id = ? AND deleted = 0`+langClause+`
		ORDER BY title
		LIMIT ? OFFSET ?`,
		args...,
	)
	return books, err
}

func (db *DB) GetBook(id int64) (*Book, error) {
	var book Book
	err := db.Get(&book, "SELECT * FROM book WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (db *DB) GetBooksByAuthor(authorID int64, limit, offset int, langs []string) ([]Book, int64, error) {
	langClause, langArgs := buildLangFilter("b", langs)

	countArgs := append([]interface{}{authorID}, langArgs...)
	var total int64
	if err := db.Get(&total, `
		SELECT COUNT(*) FROM book b
		JOIN book_author ba ON b.id = ba.book_id
		WHERE ba.author_id = ? AND b.deleted = 0`+langClause,
		countArgs...); err != nil {
		return nil, 0, err
	}

	dataArgs := append([]interface{}{authorID}, langArgs...)
	dataArgs = append(dataArgs, limit, offset)
	var books []Book
	err := db.Select(&books, `
		SELECT b.* FROM book b
		JOIN book_author ba ON b.id = ba.book_id
		WHERE ba.author_id = ? AND b.deleted = 0`+langClause+`
		ORDER BY b.title
		LIMIT ? OFFSET ?`,
		dataArgs...,
	)
	return books, total, err
}

func (db *DB) GetBooksBySeries(seriesID int64, langs []string) ([]Book, error) {
	langClause, langArgs := buildLangFilter("b", langs)
	args := append([]interface{}{seriesID}, langArgs...)
	var books []Book
	err := db.Select(&books, `
		SELECT b.* FROM book b
		JOIN book_series bs ON b.id = bs.book_id
		WHERE bs.series_id = ? AND b.deleted = 0`+langClause+`
		ORDER BY bs.seq_num, b.title`,
		args...,
	)
	return books, err
}

func (db *DB) GetBooksBySeriesPaginated(seriesID int64, limit, offset int, langs []string) ([]Book, int64, error) {
	langClause, langArgs := buildLangFilter("b", langs)

	countArgs := append([]interface{}{seriesID}, langArgs...)
	var total int64
	if err := db.Get(&total, `
		SELECT COUNT(*) FROM book b
		JOIN book_series bs ON b.id = bs.book_id
		WHERE bs.series_id = ? AND b.deleted = 0`+langClause,
		countArgs...); err != nil {
		return nil, 0, err
	}

	dataArgs := append([]interface{}{seriesID}, langArgs...)
	dataArgs = append(dataArgs, limit, offset)
	var books []Book
	err := db.Select(&books, `
		SELECT b.* FROM book b
		JOIN book_series bs ON b.id = bs.book_id
		WHERE bs.series_id = ? AND b.deleted = 0`+langClause+`
		ORDER BY bs.seq_num, b.title
		LIMIT ? OFFSET ?`,
		dataArgs...,
	)
	return books, total, err
}

func (db *DB) GetBooksByGenre(genreID int, libraryID int64, limit, offset int, langs []string) ([]Book, int64, error) {
	langClause, langArgs := buildLangFilter("b", langs)

	countArgs := append([]interface{}{genreID, libraryID}, langArgs...)
	var total int64
	if err := db.Get(&total, `
		SELECT COUNT(*) FROM book b
		JOIN book_genre bg ON b.id = bg.book_id
		WHERE bg.genre_id = ? AND b.library_id = ? AND b.deleted = 0`+langClause,
		countArgs...); err != nil {
		return nil, 0, err
	}

	dataArgs := append([]interface{}{genreID, libraryID}, langArgs...)
	dataArgs = append(dataArgs, limit, offset)
	var books []Book
	err := db.Select(&books, `
		SELECT b.* FROM book b
		JOIN book_genre bg ON b.id = bg.book_id
		WHERE bg.genre_id = ? AND b.library_id = ? AND b.deleted = 0`+langClause+`
		ORDER BY b.title
		LIMIT ? OFFSET ?`,
		dataArgs...,
	)
	return books, total, err
}

func (db *DB) SearchBooks(libraryID int64, query string, limit, offset int, langs []string) ([]Book, int64, error) {
	// Use multiple case variants for Cyrillic support since SQLite LOWER() doesn't work with Cyrillic
	pattern := "%" + query + "%"
	patternLower := "%" + strings.ToLower(query) + "%"
	patternUpper := "%" + strings.ToUpper(query) + "%"
	// Title case: first letter of each word uppercase
	patternTitle := "%" + toTitleCase(query) + "%"

	langClause, langArgs := buildLangFilter("book", langs)

	countArgs := []interface{}{libraryID, pattern, patternLower, patternUpper, patternTitle}
	countArgs = append(countArgs, langArgs...)
	var total int64
	if err := db.Get(&total, `
		SELECT COUNT(*) FROM book
		WHERE library_id = ? AND deleted = 0
		AND (title LIKE ? OR title LIKE ? OR title LIKE ? OR title LIKE ?)`+langClause,
		countArgs...); err != nil {
		return nil, 0, err
	}

	dataArgs := []interface{}{libraryID, pattern, patternLower, patternUpper, patternTitle}
	dataArgs = append(dataArgs, langArgs...)
	dataArgs = append(dataArgs, limit, offset)
	var books []Book
	err := db.Select(&books, `
		SELECT * FROM book
		WHERE library_id = ? AND deleted = 0
		AND (title LIKE ? OR title LIKE ? OR title LIKE ? OR title LIKE ?)`+langClause+`
		ORDER BY title
		LIMIT ? OFFSET ?`,
		dataArgs...,
	)
	return books, total, err
}

// toTitleCase converts the first letter of each word to uppercase (works with Cyrillic)
func toTitleCase(s string) string {
	runes := []rune(strings.ToLower(s))
	inWord := false
	for i, r := range runes {
		if r == ' ' || r == '\t' || r == '\n' {
			inWord = false
		} else if !inWord {
			runes[i] = unicode.ToUpper(r)
			inWord = true
		}
	}
	return string(runes)
}

func (db *DB) SearchAuthors(libraryID int64, query string, limit, offset int, langs []string) ([]AuthorWithCount, int64, error) {
	// Use multiple case variants for Cyrillic support
	pattern := "%" + query + "%"
	patternLower := "%" + strings.ToLower(query) + "%"
	patternUpper := "%" + strings.ToUpper(query) + "%"
	patternTitle := "%" + toTitleCase(query) + "%"

	namePatterns := []interface{}{
		pattern, patternLower, patternUpper, patternTitle,
		pattern, patternLower, patternUpper, patternTitle,
		pattern, patternLower, patternUpper, patternTitle,
	}
	nameCond := ` AND (a.first_name LIKE ? OR a.first_name LIKE ? OR a.first_name LIKE ? OR a.first_name LIKE ?
		  OR a.last_name LIKE ? OR a.last_name LIKE ? OR a.last_name LIKE ? OR a.last_name LIKE ?
		  OR a.middle_name LIKE ? OR a.middle_name LIKE ? OR a.middle_name LIKE ? OR a.middle_name LIKE ?)`

	langClause, langArgs := buildLangFilter("b", langs)

	var total int64
	if len(langs) == 0 {
		countArgs := append([]interface{}{libraryID}, namePatterns...)
		if err := db.Get(&total, `
			SELECT COUNT(DISTINCT a.id) FROM author a
			WHERE a.library_id = ?`+nameCond,
			countArgs...); err != nil {
			return nil, 0, err
		}
	} else {
		countArgs := append([]interface{}{libraryID}, namePatterns...)
		countArgs = append(countArgs, langArgs...)
		if err := db.Get(&total, `
			SELECT COUNT(DISTINCT a.id) FROM author a
			WHERE a.library_id = ?`+nameCond+`
			AND EXISTS (
				SELECT 1 FROM book_author ba
				JOIN book b ON b.id = ba.book_id
				WHERE ba.author_id = a.id AND b.deleted = 0`+langClause+`
			)`,
			countArgs...); err != nil {
			return nil, 0, err
		}
	}

	var authors []AuthorWithCount
	if len(langs) == 0 {
		dataArgs := append([]interface{}{libraryID}, namePatterns...)
		dataArgs = append(dataArgs, limit, offset)
		if err := db.Select(&authors, `
			SELECT a.*, COUNT(ba.book_id) as book_count
			FROM author a
			LEFT JOIN book_author ba ON a.id = ba.author_id
			WHERE a.library_id = ?`+nameCond+`
			GROUP BY a.id
			ORDER BY a.last_name, a.first_name
			LIMIT ? OFFSET ?`,
			dataArgs...); err != nil {
			return nil, 0, err
		}
	} else {
		langJoin := buildLangJoinCond("b", langs)
		dataArgs := append(langArgs, libraryID)
		dataArgs = append(dataArgs, namePatterns...)
		dataArgs = append(dataArgs, limit, offset)
		if err := db.Select(&authors, `
			SELECT a.*, COUNT(DISTINCT b.id) as book_count
			FROM author a
			JOIN book_author ba ON a.id = ba.author_id
			JOIN book b ON b.id = ba.book_id AND b.deleted = 0`+langJoin+`
			WHERE a.library_id = ?`+nameCond+`
			GROUP BY a.id
			ORDER BY a.last_name, a.first_name
			LIMIT ? OFFSET ?`,
			dataArgs...); err != nil {
			return nil, 0, err
		}
	}
	return authors, total, nil
}

func (db *DB) SearchSeries(libraryID int64, query string, limit, offset int, langs []string) ([]SeriesWithCount, int64, error) {
	// Use multiple case variants for Cyrillic support
	pattern := "%" + query + "%"
	patternLower := "%" + strings.ToLower(query) + "%"
	patternUpper := "%" + strings.ToUpper(query) + "%"
	patternTitle := "%" + toTitleCase(query) + "%"

	namePatterns := []interface{}{pattern, patternLower, patternUpper, patternTitle}
	nameCond := ` AND (s.name LIKE ? OR s.name LIKE ? OR s.name LIKE ? OR s.name LIKE ?)`

	langClause, langArgs := buildLangFilter("b", langs)

	var total int64
	if len(langs) == 0 {
		countArgs := append([]interface{}{libraryID}, namePatterns...)
		if err := db.Get(&total, `
			SELECT COUNT(*) FROM series s
			WHERE s.library_id = ?`+nameCond,
			countArgs...); err != nil {
			return nil, 0, err
		}
	} else {
		countArgs := append([]interface{}{libraryID}, namePatterns...)
		countArgs = append(countArgs, langArgs...)
		if err := db.Get(&total, `
			SELECT COUNT(DISTINCT s.id) FROM series s
			WHERE s.library_id = ?`+nameCond+`
			AND EXISTS (
				SELECT 1 FROM book_series bs
				JOIN book b ON b.id = bs.book_id
				WHERE bs.series_id = s.id AND b.deleted = 0`+langClause+`
			)`,
			countArgs...); err != nil {
			return nil, 0, err
		}
	}

	var series []SeriesWithCount
	if len(langs) == 0 {
		dataArgs := append([]interface{}{libraryID}, namePatterns...)
		dataArgs = append(dataArgs, limit, offset)
		if err := db.Select(&series, `
			SELECT s.*, COUNT(bs.book_id) as book_count
			FROM series s
			LEFT JOIN book_series bs ON s.id = bs.series_id
			WHERE s.library_id = ?`+nameCond+`
			GROUP BY s.id
			ORDER BY s.name
			LIMIT ? OFFSET ?`,
			dataArgs...); err != nil {
			return nil, 0, err
		}
	} else {
		langJoin := buildLangJoinCond("b", langs)
		dataArgs := append(langArgs, libraryID)
		dataArgs = append(dataArgs, namePatterns...)
		dataArgs = append(dataArgs, limit, offset)
		if err := db.Select(&series, `
			SELECT s.*, COUNT(DISTINCT b.id) as book_count
			FROM series s
			JOIN book_series bs ON s.id = bs.series_id
			JOIN book b ON b.id = bs.book_id AND b.deleted = 0`+langJoin+`
			WHERE s.library_id = ?`+nameCond+`
			GROUP BY s.id
			ORDER BY s.name
			LIMIT ? OFFSET ?`,
			dataArgs...); err != nil {
			return nil, 0, err
		}
	}
	return series, total, nil
}

func (db *DB) GetBookAuthors(bookID int64) ([]Author, error) {
	var authors []Author
	err := db.Select(&authors, `
		SELECT a.* FROM author a
		JOIN book_author ba ON a.id = ba.author_id
		WHERE ba.book_id = ?`,
		bookID,
	)
	return authors, err
}

func (db *DB) GetBookSeries(bookID int64) ([]Series, error) {
	var series []Series
	err := db.Select(&series, `
		SELECT s.* FROM series s
		JOIN book_series bs ON s.id = bs.series_id
		WHERE bs.book_id = ?`,
		bookID,
	)
	return series, err
}

func (db *DB) GetBookGenres(bookID int64) ([]Genre, error) {
	var genres []Genre
	err := db.Select(&genres, `
		SELECT g.* FROM genre g
		JOIN book_genre bg ON g.id = bg.genre_id
		WHERE bg.book_id = ?`,
		bookID,
	)
	return genres, err
}

// User queries

func (db *DB) GetUserByUsername(username string) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT * FROM user WHERE username = ?", username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *DB) GetUserByID(id int64) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT * FROM user WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *DB) GetUsers() ([]User, error) {
	var users []User
	err := db.Select(&users, "SELECT * FROM user ORDER BY username")
	return users, err
}

func (db *DB) CountUsers() (int64, error) {
	var count int64
	err := db.Get(&count, "SELECT COUNT(*) FROM user")
	return count, err
}

func (db *DB) CreateUser(user *User) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO user (username, password_hash, role)
		VALUES (?, ?, ?)`,
		user.Username, user.PasswordHash, user.Role,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) UpdateUserPassword(userID int64, passwordHash string) error {
	_, err := db.Exec(`
		UPDATE user SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		passwordHash, userID,
	)
	return err
}

func (db *DB) UpdateUserRole(userID int64, role string) error {
	_, err := db.Exec(`
		UPDATE user SET role = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		role, userID,
	)
	return err
}

func (db *DB) DeleteUser(userID int64) error {
	_, err := db.Exec("DELETE FROM user WHERE id = ?", userID)
	return err
}

// Session queries

func (db *DB) GetSession(sessionID string) (*Session, error) {
	var session Session
	err := db.Get(&session, "SELECT * FROM session WHERE id = ?", sessionID)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (db *DB) CreateSession(session *Session) error {
	_, err := db.Exec(`
		INSERT INTO session (id, user_id, expires_at)
		VALUES (?, ?, ?)`,
		session.ID, session.UserID, session.ExpiresAt,
	)
	return err
}

func (db *DB) DeleteSession(sessionID string) error {
	_, err := db.Exec("DELETE FROM session WHERE id = ?", sessionID)
	return err
}

func (db *DB) DeleteExpiredSessions() error {
	_, err := db.Exec("DELETE FROM session WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

func (db *DB) DeleteUserSessions(userID int64) error {
	_, err := db.Exec("DELETE FROM session WHERE user_id = ?", userID)
	return err
}

// OIDC Session queries

func (db *DB) GetOIDCSession(sessionID string) (*OIDCSession, error) {
	var session OIDCSession
	err := db.Get(&session, "SELECT * FROM oidc_session WHERE id = ?", sessionID)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (db *DB) CreateOIDCSession(session *OIDCSession) error {
	_, err := db.Exec(`
		INSERT INTO oidc_session (id, username, role, id_token, access_token, refresh_token, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.Username, session.Role, session.IDToken, session.AccessToken, session.RefreshToken, session.ExpiresAt,
	)
	return err
}

func (db *DB) DeleteOIDCSession(sessionID string) error {
	_, err := db.Exec("DELETE FROM oidc_session WHERE id = ?", sessionID)
	return err
}

func (db *DB) DeleteExpiredOIDCSessions() error {
	_, err := db.Exec("DELETE FROM oidc_session WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

// Language settings queries

// GetAvailableLanguages returns all distinct non-empty language codes for a specific library.
func (db *DB) GetAvailableLanguages(libraryID int64) ([]string, error) {
	var langs []string
	err := db.Select(&langs, "SELECT DISTINCT lang FROM book WHERE library_id = ? AND lang != '' AND deleted = 0 ORDER BY lang", libraryID)
	if err != nil {
		return nil, err
	}
	if langs == nil {
		langs = []string{}
	}
	return langs, nil
}

// GetLibraryLangFilter returns the language filter for a specific library.
// Returns empty slice if not configured (meaning show all).
func (db *DB) GetLibraryLangFilter(libraryID int64) ([]string, error) {
	var value string
	err := db.Get(&value, "SELECT lang_filter FROM library WHERE id = ?", libraryID)
	if err != nil {
		return []string{}, nil
	}
	var langs []string
	if err := json.Unmarshal([]byte(value), &langs); err != nil {
		return []string{}, nil
	}
	if langs == nil {
		langs = []string{}
	}
	return langs, nil
}

// SaveLibraryLangFilter stores the language filter for a specific library.
func (db *DB) SaveLibraryLangFilter(libraryID int64, langs []string) error {
	if langs == nil {
		langs = []string{}
	}
	data, err := json.Marshal(langs)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		UPDATE library SET lang_filter = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		string(data), libraryID,
	)
	return err
}
