package db

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"
)

// recorder is a SQLite driver that remembers every statement it prepares, so
// that tests can inspect the plans of the exact SQL the query methods run.
type recorder struct {
	sqlite3.SQLiteDriver
	mu      sync.Mutex
	queries []string
}

func (r *recorder) Open(dsn string) (driver.Conn, error) {
	conn, err := r.SQLiteDriver.Open(dsn)
	if err != nil {
		return nil, err
	}
	return &recordingConn{Conn: conn, recorder: r}, nil
}

func (r *recorder) take() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	queries := r.queries
	r.queries = nil
	return queries
}

// recordingConn exposes only driver.Conn, which makes database/sql send every
// statement through Prepare.
type recordingConn struct {
	driver.Conn
	recorder *recorder
}

func (c *recordingConn) Prepare(query string) (driver.Stmt, error) {
	c.recorder.mu.Lock()
	c.recorder.queries = append(c.recorder.queries, query)
	c.recorder.mu.Unlock()
	return c.Conn.Prepare(query)
}

var planRecorder = &recorder{}

func init() {
	sql.Register("sqlite3_recorder", planRecorder)
}

func queryPlan(t *testing.T, database *DB, query string) string {
	t.Helper()
	args := make([]interface{}, strings.Count(query, "?"))
	for i := range args {
		args[i] = 1
	}
	var steps []struct {
		ID      int    `db:"id"`
		Parent  int    `db:"parent"`
		NotUsed int    `db:"notused"`
		Detail  string `db:"detail"`
	}
	if err := database.Select(&steps, "EXPLAIN QUERY PLAN "+query, args...); err != nil {
		t.Fatalf("explain %s: %v", query, err)
	}
	var plan strings.Builder
	for _, s := range steps {
		plan.WriteString(s.Detail + "\n")
	}
	return plan.String()
}

func closeDB(t *testing.T, database *DB) {
	t.Helper()
	if err := database.Close(); err != nil {
		t.Errorf("close database: %v", err)
	}
}

func TestLangFilter_JoinedQueriesDoNotUseLangIndex(t *testing.T) {
	// Migrate on a normal connection: Prepare runs only the first statement of a script.
	path := t.TempDir() + "/plan.db"
	setup, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { closeDB(t, setup) })
	if err := setup.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	conn, err := sql.Open("sqlite3_recorder", path)
	if err != nil {
		t.Fatalf("open recorder: %v", err)
	}
	database := &DB{sqlx.NewDb(conn, "sqlite3")}
	t.Cleanup(func() { closeDB(t, database) })

	for _, langs := range [][]string{{"ru"}, {"ru", "en"}} {
		calls := map[string]func() error{
			"GetAuthors":           func() error { _, err := database.GetAuthors(1, 10, 0, langs); return err },
			"GetAuthorsFiltered":   func() error { _, err := database.GetAuthorsFiltered(1, "a", 10, 0, langs); return err },
			"GetAuthorsByLetter":   func() error { _, err := database.GetAuthorsByLetter(1, "a", langs); return err },
			"CountAuthorsByPrefix": func() error { _, err := database.CountAuthorsByPrefix(1, "a", langs); return err },
			"SearchAuthors":        func() error { _, _, err := database.SearchAuthors(1, "a", 10, 0, langs); return err },
			"GetSeries":            func() error { _, _, err := database.GetSeries(1, 10, 0, langs); return err },
			"GetSeriesFiltered":    func() error { _, err := database.GetSeriesFiltered(1, "a", 10, 0, langs); return err },
			"SearchSeries":         func() error { _, _, err := database.SearchSeries(1, "a", 10, 0, langs); return err },
			"GetBooksByAuthor":     func() error { _, _, err := database.GetBooksByAuthor(1, 10, 0, langs); return err },
			"GetBooksBySeries":     func() error { _, err := database.GetBooksBySeries(1, langs); return err },
			"GetBooksBySeriesPaginated": func() error {
				_, _, err := database.GetBooksBySeriesPaginated(1, 10, 0, langs)
				return err
			},
			"GetBooksByGenre": func() error { _, _, err := database.GetBooksByGenre(1, 1, 10, 0, langs); return err },
		}

		for name, call := range calls {
			planRecorder.take()
			if err := call(); err != nil {
				t.Fatalf("%s %v: %v", name, langs, err)
			}
			queries := planRecorder.take()
			filtered := 0
			for _, query := range queries {
				if !strings.Contains(query, "b.lang IN") {
					continue
				}
				filtered++
				if plan := queryPlan(t, database, query); strings.Contains(plan, "idx_book_lang") {
					t.Errorf("%s %v uses idx_book_lang:\n%s\n%s", name, langs, plan, query)
				}
			}
			if filtered == 0 {
				t.Errorf("%s %v ran no language-filtered query", name, langs)
			}
		}
	}
}

func TestLangFilter_JoinedQueryResults(t *testing.T) {
	database := newTestDB(t)
	database.SetMaxOpenConns(1)
	libID := seedLibrary(t, database)

	alpha := seedAuthor(t, database, libID, "Alpha", "Ann")
	beta := seedAuthor(t, database, libID, "Beta", "Bob")
	gamma := seedAuthor(t, database, libID, "Gamma", "Gil")
	saga := seedSeries(t, database, libID, "Saga")
	if _, err := database.Exec(`INSERT INTO genre (id, name) VALUES (1, 'Genre')`); err != nil {
		t.Fatalf("seed genre: %v", err)
	}

	add := func(title, lang string, author int64) int64 {
		id := seedBook(t, database, libID, title, lang)
		linkBookAuthor(t, database, id, author)
		linkBookSeries(t, database, id, saga)
		if _, err := database.Exec(`INSERT INTO book_genre (book_id, genre_id) VALUES (?, 1)`, id); err != nil {
			t.Fatalf("link genre: %v", err)
		}
		return id
	}
	add("B ru", "ru", alpha)
	add("A en", "en", alpha)
	add("C de", "de", beta)
	deleted := add("D deleted", "ru", gamma)
	if _, err := database.Exec(`UPDATE book SET deleted = 1 WHERE id = ?`, deleted); err != nil {
		t.Fatalf("delete book: %v", err)
	}

	tests := []struct {
		langs   []string
		authors map[string]int
		titles  []string
	}{
		{[]string{"ru"}, map[string]int{"Alpha": 1}, []string{"B ru"}},
		{[]string{"ru", "en"}, map[string]int{"Alpha": 2}, []string{"A en", "B ru"}},
		{[]string{"de", "ru"}, map[string]int{"Alpha": 1, "Beta": 1}, []string{"B ru", "C de"}},
		{[]string{"fr"}, map[string]int{}, []string{}},
	}

	for _, tc := range tests {
		search, total, err := database.SearchAuthors(libID, "a", 10, 0, tc.langs)
		if err != nil {
			t.Fatalf("SearchAuthors %v: %v", tc.langs, err)
		}
		filtered, err := database.GetAuthorsFiltered(libID, "a", 10, 0, tc.langs)
		if err != nil {
			t.Fatalf("GetAuthorsFiltered %v: %v", tc.langs, err)
		}
		for name, authors := range map[string][]AuthorWithCount{"SearchAuthors": search, "GetAuthorsFiltered": filtered.Authors} {
			got := map[string]int{}
			for _, a := range authors {
				got[a.LastName] = a.BookCount
			}
			if !reflect.DeepEqual(got, tc.authors) {
				t.Errorf("%s %v: got %v, want %v", name, tc.langs, got, tc.authors)
			}
		}
		if int(total) != len(tc.authors) || filtered.Total != len(tc.authors) {
			t.Errorf("author totals %v: got %d and %d, want %d", tc.langs, total, filtered.Total, len(tc.authors))
		}

		bySeries, seriesTotal, err := database.GetBooksBySeriesPaginated(saga, 10, 0, tc.langs)
		if err != nil {
			t.Fatalf("GetBooksBySeriesPaginated %v: %v", tc.langs, err)
		}
		byGenre, genreTotal, err := database.GetBooksByGenre(1, libID, 10, 0, tc.langs)
		if err != nil {
			t.Fatalf("GetBooksByGenre %v: %v", tc.langs, err)
		}
		for name, books := range map[string][]Book{"series": bySeries, "genre": byGenre} {
			titles := []string{}
			for _, b := range books {
				titles = append(titles, b.Title)
			}
			if !reflect.DeepEqual(titles, tc.titles) {
				t.Errorf("books by %s %v: got %v, want %v", name, tc.langs, titles, tc.titles)
			}
		}
		if int(seriesTotal) != len(tc.titles) || int(genreTotal) != len(tc.titles) {
			t.Errorf("book totals %v: got %d and %d, want %d", tc.langs, seriesTotal, genreTotal, len(tc.titles))
		}
	}
}
