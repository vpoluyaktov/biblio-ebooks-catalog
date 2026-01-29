-- Libraries
CREATE TABLE IF NOT EXISTS library (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    inpx TEXT DEFAULT '',
    version TEXT DEFAULT '',
    first_author INTEGER DEFAULT 0,
    without_deleted INTEGER DEFAULT 0,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Authors
CREATE TABLE IF NOT EXISTS author (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL,
    last_name TEXT DEFAULT '',
    first_name TEXT DEFAULT '',
    middle_name TEXT DEFAULT '',
    FOREIGN KEY (library_id) REFERENCES library(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_author_library ON author(library_id);
CREATE INDEX IF NOT EXISTS idx_author_name ON author(last_name, first_name);

-- Series
CREATE TABLE IF NOT EXISTS series (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (library_id) REFERENCES library(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_series_library ON series(library_id);
CREATE INDEX IF NOT EXISTS idx_series_name ON series(name);

-- Genres
CREATE TABLE IF NOT EXISTS genre (
    id INTEGER PRIMARY KEY,
    parent_id INTEGER DEFAULT 0,
    name TEXT NOT NULL,
    code TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_genre_parent ON genre(parent_id);
CREATE INDEX IF NOT EXISTS idx_genre_code ON genre(code);

-- Books
CREATE TABLE IF NOT EXISTS book (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    lang TEXT DEFAULT '',
    file TEXT NOT NULL,
    archive TEXT DEFAULT '',
    format TEXT DEFAULT 'fb2',
    size INTEGER DEFAULT 0,
    rating INTEGER DEFAULT 0,
    deleted INTEGER DEFAULT 0,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    id_in_lib INTEGER DEFAULT 0,
    first_author_id INTEGER DEFAULT NULL,
    keywords TEXT DEFAULT '',
    FOREIGN KEY (library_id) REFERENCES library(id) ON DELETE CASCADE,
    FOREIGN KEY (first_author_id) REFERENCES author(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_book_library ON book(library_id);
CREATE INDEX IF NOT EXISTS idx_book_title ON book(title);
CREATE INDEX IF NOT EXISTS idx_book_lang ON book(lang);
CREATE INDEX IF NOT EXISTS idx_book_format ON book(format);
CREATE INDEX IF NOT EXISTS idx_book_first_author ON book(first_author_id);

-- Book-Author relationship (many-to-many)
CREATE TABLE IF NOT EXISTS book_author (
    book_id INTEGER NOT NULL,
    author_id INTEGER NOT NULL,
    PRIMARY KEY (book_id, author_id),
    FOREIGN KEY (book_id) REFERENCES book(id) ON DELETE CASCADE,
    FOREIGN KEY (author_id) REFERENCES author(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_book_author_book ON book_author(book_id);
CREATE INDEX IF NOT EXISTS idx_book_author_author ON book_author(author_id);

-- Book-Series relationship (many-to-many with sequence number)
CREATE TABLE IF NOT EXISTS book_series (
    book_id INTEGER NOT NULL,
    series_id INTEGER NOT NULL,
    seq_num INTEGER DEFAULT 0,
    PRIMARY KEY (book_id, series_id),
    FOREIGN KEY (book_id) REFERENCES book(id) ON DELETE CASCADE,
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_book_series_book ON book_series(book_id);
CREATE INDEX IF NOT EXISTS idx_book_series_series ON book_series(series_id);

-- Book-Genre relationship (many-to-many)
CREATE TABLE IF NOT EXISTS book_genre (
    book_id INTEGER NOT NULL,
    genre_id INTEGER NOT NULL,
    PRIMARY KEY (book_id, genre_id),
    FOREIGN KEY (book_id) REFERENCES book(id) ON DELETE CASCADE,
    FOREIGN KEY (genre_id) REFERENCES genre(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_book_genre_book ON book_genre(book_id);
CREATE INDEX IF NOT EXISTS idx_book_genre_genre ON book_genre(genre_id);

-- Users
CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'readonly',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_username ON user(username);

-- Sessions
CREATE TABLE IF NOT EXISTS session (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_user ON session(user_id);
CREATE INDEX IF NOT EXISTS idx_session_expires ON session(expires_at);

-- OIDC Sessions (for OAuth2/OIDC token storage)
CREATE TABLE IF NOT EXISTS oidc_session (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    role TEXT NOT NULL,
    id_token TEXT,
    access_token TEXT,
    refresh_token TEXT,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_oidc_session_expires ON oidc_session(expires_at);
