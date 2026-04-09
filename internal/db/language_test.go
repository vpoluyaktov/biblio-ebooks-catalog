package db

import (
	"testing"
)

// newTestDB creates an in-memory SQLite database for testing.
func newTestDB(t *testing.T) *DB {
	t.Helper()
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// seedLibrary inserts a test library and returns its ID.
func seedLibrary(t *testing.T, database *DB) int64 {
	t.Helper()
	lib := &Library{
		Name:    "Test Library",
		Path:    "/tmp/testbooks",
		Enabled: true,
	}
	id, err := database.CreateLibrary(lib)
	if err != nil {
		t.Fatalf("failed to create test library: %v", err)
	}
	return id
}

// seedBook inserts a minimal book and returns its ID.
func seedBook(t *testing.T, database *DB, libID int64, title, lang string) int64 {
	t.Helper()
	result, err := database.Exec(
		`INSERT INTO book (library_id, title, lang, file, format, deleted) VALUES (?, ?, ?, ?, ?, 0)`,
		libID, title, lang, "file.fb2", "fb2",
	)
	if err != nil {
		t.Fatalf("seedBook %q (%s): %v", title, lang, err)
	}
	id, _ := result.LastInsertId()
	return id
}

// seedAuthor inserts an author and returns its ID.
func seedAuthor(t *testing.T, database *DB, libID int64, lastName, firstName string) int64 {
	t.Helper()
	result, err := database.Exec(
		`INSERT INTO author (library_id, last_name, first_name) VALUES (?, ?, ?)`,
		libID, lastName, firstName,
	)
	if err != nil {
		t.Fatalf("seedAuthor %q: %v", lastName, err)
	}
	id, _ := result.LastInsertId()
	return id
}

// linkBookAuthor links a book and an author.
func linkBookAuthor(t *testing.T, database *DB, bookID, authorID int64) {
	t.Helper()
	_, err := database.Exec(`INSERT INTO book_author (book_id, author_id) VALUES (?, ?)`, bookID, authorID)
	if err != nil {
		t.Fatalf("linkBookAuthor book=%d author=%d: %v", bookID, authorID, err)
	}
}

// seedSeries inserts a series and returns its ID.
func seedSeries(t *testing.T, database *DB, libID int64, name string) int64 {
	t.Helper()
	result, err := database.Exec(
		`INSERT INTO series (library_id, name) VALUES (?, ?)`,
		libID, name,
	)
	if err != nil {
		t.Fatalf("seedSeries %q: %v", name, err)
	}
	id, _ := result.LastInsertId()
	return id
}

// linkBookSeries links a book to a series.
func linkBookSeries(t *testing.T, database *DB, bookID, seriesID int64) {
	t.Helper()
	_, err := database.Exec(`INSERT INTO book_series (book_id, series_id, seq_num) VALUES (?, ?, 1)`, bookID, seriesID)
	if err != nil {
		t.Fatalf("linkBookSeries book=%d series=%d: %v", bookID, seriesID, err)
	}
}

// containsString checks if a string slice contains a value.
func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// ---- GetAvailableLanguages ----

func TestGetAvailableLanguages_Empty(t *testing.T) {
	database := newTestDB(t)
	libID := seedLibrary(t, database)
	// No books inserted.
	_ = libID

	langs, err := database.GetAvailableLanguages()
	if err != nil {
		t.Fatalf("GetAvailableLanguages error: %v", err)
	}
	// No books → empty slice (not nil error, just empty result).
	if len(langs) != 0 {
		t.Errorf("expected empty slice, got %v", langs)
	}
}

func TestGetAvailableLanguages_ReturnsDistinctNonEmpty(t *testing.T) {
	database := newTestDB(t)
	libID := seedLibrary(t, database)

	seedBook(t, database, libID, "Русская книга 1", "ru")
	seedBook(t, database, libID, "Русская книга 2", "ru") // duplicate — should appear once
	seedBook(t, database, libID, "English book", "en")
	seedBook(t, database, libID, "German book", "de")
	// A book with empty lang — must NOT appear.
	seedBook(t, database, libID, "No-lang book", "")
	// A deleted book — must NOT appear.
	_, err := database.Exec(
		`INSERT INTO book (library_id, title, lang, file, format, deleted) VALUES (?, ?, ?, ?, ?, 1)`,
		libID, "Deleted book", "fr", "del.fb2", "fb2",
	)
	if err != nil {
		t.Fatalf("insert deleted book: %v", err)
	}

	langs, err := database.GetAvailableLanguages()
	if err != nil {
		t.Fatalf("GetAvailableLanguages error: %v", err)
	}

	// Expect exactly "de", "en", "ru" — distinct, non-empty, non-deleted.
	expected := map[string]bool{"ru": true, "en": true, "de": true}
	if len(langs) != len(expected) {
		t.Errorf("expected %d languages, got %d: %v", len(expected), len(langs), langs)
	}
	for _, l := range langs {
		if !expected[l] {
			t.Errorf("unexpected language %q in result", l)
		}
	}
}

// ---- GetSelectedLanguages / SaveSelectedLanguages ----

func TestGetSelectedLanguages_NotSet(t *testing.T) {
	database := newTestDB(t)

	langs, err := database.GetSelectedLanguages()
	if err != nil {
		t.Fatalf("GetSelectedLanguages error: %v", err)
	}
	if len(langs) != 0 {
		t.Errorf("expected empty slice when not set, got %v", langs)
	}
}

func TestSaveAndGetSelectedLanguages_RoundTrip(t *testing.T) {
	database := newTestDB(t)

	want := []string{"ru", "en"}
	if err := database.SaveSelectedLanguages(want); err != nil {
		t.Fatalf("SaveSelectedLanguages error: %v", err)
	}

	got, err := database.GetSelectedLanguages()
	if err != nil {
		t.Fatalf("GetSelectedLanguages error: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	wantSet := map[string]bool{"ru": true, "en": true}
	for _, l := range got {
		if !wantSet[l] {
			t.Errorf("unexpected language %q in result", l)
		}
	}
}

func TestSaveSelectedLanguages_Overwrites(t *testing.T) {
	database := newTestDB(t)

	// Save first value.
	if err := database.SaveSelectedLanguages([]string{"ru", "en", "de"}); err != nil {
		t.Fatalf("first SaveSelectedLanguages error: %v", err)
	}
	// Overwrite with different value.
	if err := database.SaveSelectedLanguages([]string{"fr"}); err != nil {
		t.Fatalf("second SaveSelectedLanguages error: %v", err)
	}

	got, err := database.GetSelectedLanguages()
	if err != nil {
		t.Fatalf("GetSelectedLanguages error: %v", err)
	}
	if len(got) != 1 || got[0] != "fr" {
		t.Errorf("expected [fr], got %v", got)
	}
}

func TestSaveSelectedLanguages_EmptySlice(t *testing.T) {
	database := newTestDB(t)

	// First set something.
	if err := database.SaveSelectedLanguages([]string{"ru"}); err != nil {
		t.Fatalf("SaveSelectedLanguages error: %v", err)
	}
	// Then clear with empty slice.
	if err := database.SaveSelectedLanguages([]string{}); err != nil {
		t.Fatalf("SaveSelectedLanguages(empty) error: %v", err)
	}

	got, err := database.GetSelectedLanguages()
	if err != nil {
		t.Fatalf("GetSelectedLanguages error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice after clearing, got %v", got)
	}
}

// ---- GetAuthorsFiltered with language filter ----

func TestGetAuthorsFiltered_LangFilter_ExcludesAuthorsWithoutMatchingBooks(t *testing.T) {
	database := newTestDB(t)
	libID := seedLibrary(t, database)

	// Author A has only English books.
	authorA := seedAuthor(t, database, libID, "Smith", "John")
	bookEN := seedBook(t, database, libID, "English Book", "en")
	linkBookAuthor(t, database, bookEN, authorA)

	// Author B has only Russian books.
	authorB := seedAuthor(t, database, libID, "Иванов", "Иван")
	bookRU := seedBook(t, database, libID, "Русская книга", "ru")
	linkBookAuthor(t, database, bookRU, authorB)

	// Filter to Russian only — Author A should be excluded.
	result, err := database.GetAuthorsFiltered(libID, "", 50, 0, []string{"ru"})
	if err != nil {
		t.Fatalf("GetAuthorsFiltered error: %v", err)
	}

	for _, a := range result.Authors {
		if a.ID == authorA {
			t.Errorf("author %q (English only) should not appear in Russian filter results", "Smith")
		}
	}

	found := false
	for _, a := range result.Authors {
		if a.ID == authorB {
			found = true
		}
	}
	if !found {
		t.Errorf("author %q should appear in Russian filter results", "Иванов")
	}
}

func TestGetAuthorsFiltered_EmptyLang_NoFiltering(t *testing.T) {
	database := newTestDB(t)
	libID := seedLibrary(t, database)

	authorA := seedAuthor(t, database, libID, "Smith", "John")
	bookEN := seedBook(t, database, libID, "English Book", "en")
	linkBookAuthor(t, database, bookEN, authorA)

	authorB := seedAuthor(t, database, libID, "Иванов", "Иван")
	bookRU := seedBook(t, database, libID, "Русская книга", "ru")
	linkBookAuthor(t, database, bookRU, authorB)

	// Empty langs — no filtering, both authors should appear.
	result, err := database.GetAuthorsFiltered(libID, "", 50, 0, []string{})
	if err != nil {
		t.Fatalf("GetAuthorsFiltered error: %v", err)
	}

	foundA, foundB := false, false
	for _, a := range result.Authors {
		if a.ID == authorA {
			foundA = true
		}
		if a.ID == authorB {
			foundB = true
		}
	}
	if !foundA {
		t.Errorf("expected author A to appear when no language filter is set")
	}
	if !foundB {
		t.Errorf("expected author B to appear when no language filter is set")
	}
}

func TestGetAuthorsFiltered_LangFilter_AllBooksFiltered_EmptyResult(t *testing.T) {
	database := newTestDB(t)
	libID := seedLibrary(t, database)

	// All books are in Russian.
	authorA := seedAuthor(t, database, libID, "Петров", "Пётр")
	bookRU := seedBook(t, database, libID, "Русская книга", "ru")
	linkBookAuthor(t, database, bookRU, authorA)

	// Filter to English — no authors should appear.
	result, err := database.GetAuthorsFiltered(libID, "", 50, 0, []string{"en"})
	if err != nil {
		t.Fatalf("GetAuthorsFiltered error: %v", err)
	}
	if len(result.Authors) != 0 {
		t.Errorf("expected no authors, got %v", result.Authors)
	}
}

// ---- GetSeriesFiltered with language filter ----

func TestGetSeriesFiltered_LangFilter_ExcludesSeriesWithoutMatchingBooks(t *testing.T) {
	database := newTestDB(t)
	libID := seedLibrary(t, database)

	// Series A contains only English books.
	seriesA := seedSeries(t, database, libID, "English Series")
	bookEN := seedBook(t, database, libID, "English Book", "en")
	linkBookSeries(t, database, bookEN, seriesA)

	// Series B contains only Russian books.
	seriesB := seedSeries(t, database, libID, "Русская серия")
	bookRU := seedBook(t, database, libID, "Русская книга", "ru")
	linkBookSeries(t, database, bookRU, seriesB)

	// Filter to English only — Series B should be excluded.
	result, err := database.GetSeriesFiltered(libID, "", 50, 0, []string{"en"})
	if err != nil {
		t.Fatalf("GetSeriesFiltered error: %v", err)
	}

	for _, s := range result.Series {
		if s.ID == seriesB {
			t.Errorf("Russian-only series should not appear in English filter results")
		}
	}

	found := false
	for _, s := range result.Series {
		if s.ID == seriesA {
			found = true
		}
	}
	if !found {
		t.Errorf("English series should appear in English filter results")
	}
}

func TestGetSeriesFiltered_EmptyLang_NoFiltering(t *testing.T) {
	database := newTestDB(t)
	libID := seedLibrary(t, database)

	seriesA := seedSeries(t, database, libID, "English Series")
	bookEN := seedBook(t, database, libID, "English Book", "en")
	linkBookSeries(t, database, bookEN, seriesA)

	seriesB := seedSeries(t, database, libID, "Русская серия")
	bookRU := seedBook(t, database, libID, "Русская книга", "ru")
	linkBookSeries(t, database, bookRU, seriesB)

	result, err := database.GetSeriesFiltered(libID, "", 50, 0, []string{})
	if err != nil {
		t.Fatalf("GetSeriesFiltered error: %v", err)
	}

	foundA, foundB := false, false
	for _, s := range result.Series {
		if s.ID == seriesA {
			foundA = true
		}
		if s.ID == seriesB {
			foundB = true
		}
	}
	if !foundA {
		t.Errorf("expected series A when no language filter is set")
	}
	if !foundB {
		t.Errorf("expected series B when no language filter is set")
	}
}

// ---- SearchBooks with language filter ----

func TestSearchBooks_LangFilter_OnlyReturnsMatchingLanguage(t *testing.T) {
	database := newTestDB(t)
	libID := seedLibrary(t, database)

	seedBook(t, database, libID, "Magic Book", "en")
	seedBook(t, database, libID, "Magic Kniga", "ru")

	books, total, err := database.SearchBooks(libID, "Magic", 50, 0, []string{"en"})
	if err != nil {
		t.Fatalf("SearchBooks error: %v", err)
	}

	if total == 0 || len(books) == 0 {
		t.Fatal("expected at least one English book in results")
	}
	for _, b := range books {
		if b.Lang != "en" {
			t.Errorf("expected only English books, got lang=%q for book %q", b.Lang, b.Title)
		}
	}
}

func TestSearchBooks_EmptyLang_NoFiltering(t *testing.T) {
	database := newTestDB(t)
	libID := seedLibrary(t, database)

	seedBook(t, database, libID, "Magic Book EN", "en")
	seedBook(t, database, libID, "Magic Book RU", "ru")

	_, total, err := database.SearchBooks(libID, "Magic", 50, 0, []string{})
	if err != nil {
		t.Fatalf("SearchBooks error: %v", err)
	}
	if total < 2 {
		t.Errorf("expected both books with no language filter, got total=%d", total)
	}
}

func TestSearchBooks_LangFilter_NoMatch_EmptyResult(t *testing.T) {
	database := newTestDB(t)
	libID := seedLibrary(t, database)

	// All books in Russian.
	seedBook(t, database, libID, "Magic Kniga", "ru")

	books, total, err := database.SearchBooks(libID, "Magic", 50, 0, []string{"en"})
	if err != nil {
		t.Fatalf("SearchBooks error: %v", err)
	}
	if total != 0 || len(books) != 0 {
		t.Errorf("expected no results when filtering to English and all books are Russian, got total=%d", total)
	}
}
