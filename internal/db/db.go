package db

import (
	"embed"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

//go:embed genres_en.tsv
var genresFS embed.FS

type DB struct {
	*sqlx.DB
}

type Tx struct {
	*sqlx.Tx
}

func Open(path string) (*DB, error) {
	db, err := sqlx.Open("sqlite3", path+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Migrate() error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	_, err = db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Add lang_filter column to existing library tables (ignore error if already exists)
	db.Exec("ALTER TABLE library ADD COLUMN lang_filter TEXT NOT NULL DEFAULT '[]'")

	return nil
}

func (db *DB) LoadGenres() error {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM genre")
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	data, err := genresFS.ReadFile("genres_en.tsv")
	if err != nil {
		return fmt.Errorf("failed to read genres: %w", err)
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		var parentID, id int
		fmt.Sscanf(parts[0], "%d", &parentID)
		fmt.Sscanf(parts[1], "%d", &id)
		name := parts[2]
		code := ""
		if len(parts) > 3 {
			code = parts[3]
		}

		_, err := tx.Exec(
			"INSERT OR IGNORE INTO genre (id, parent_id, name, code) VALUES (?, ?, ?, ?)",
			id, parentID, name, code,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
