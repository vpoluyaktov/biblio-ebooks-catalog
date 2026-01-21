# opds-server Specification

A Go-based web server for managing e-book libraries, ported from the freeLib Qt/C++ application.

## Table of Contents

1. [Project Overview](#project-overview)
2. [Original freeLib Analysis](#original-freelib-analysis)
3. [Target Architecture](#target-architecture)
4. [Data Models](#data-models)
5. [API Endpoints](#api-endpoints)
6. [Implementation Plan](#implementation-plan)

---

## Project Overview

### Goal

Port the freeLib desktop application (Qt/C++) to a Go-based web server with a modern web interface. The new application will provide:

- **Web-based UI** for browsing and managing e-book libraries
- **OPDS/OPDS2 catalog server** for e-reader compatibility
- **REST API** for programmatic access
- **Multi-library support** with SQLite database backend
- **Book format support**: FB2, EPUB, MOBI, AZW3, PDF, DJVU

### Key Features to Port

1. Library management (create, import from INPX, update)
2. Book cataloging with metadata (authors, series, genres, tags)
3. Search and filtering (by author, title, series, genre, language)
4. OPDS/OPDS2 server for e-reader access
5. Book format conversion (FB2 → EPUB, MOBI, AZW3)
6. Cover image extraction and caching
7. Book annotation/description extraction

---

## Original freeLib Analysis

### Technology Stack (Original)

- **Language**: C++17/20
- **Framework**: Qt 6 (Qt 5 compatible)
- **Database**: SQLite
- **HTTP Server**: QHttpServer
- **Archive Support**: QuaZip (ZIP handling)
- **Build System**: CMake

### Source Code Structure

```
src/
├── main.cpp              # Application entry, CLI handling
├── mainwindow.cpp/h      # Main GUI window (Qt Widgets)
├── library.cpp/h         # Library data structures (SLib, SBook, SAuthor, SSeries)
├── opds_server.cpp/h     # OPDS/OPDS2/HTML server implementation
├── importthread.cpp/h    # Library import from INPX/FB2/EPUB files
├── exportthread.cpp/h    # Book export and conversion
├── bookfile.cpp/h        # Book file handling, cover/annotation extraction
├── epubreader.cpp/h      # EPUB format parser
├── utilites.cpp/h        # Utility functions, database operations
├── options.cpp/h         # Application settings and export options
├── genre.tsv             # Genre definitions (347 genres in hierarchy)
└── xsl/                  # XSL transformations, CSS, fonts
    ├── opds/             # OPDS web assets (CSS, JS, SVG)
    ├── fonts/            # Embedded fonts
    └── css/              # Stylesheets
```

### Database Schema

The application uses SQLite with the following tables:

#### Core Tables

| Table | Description |
|-------|-------------|
| `lib` | Libraries (id, name, path, inpx, version, firstauthor, woDeleted) |
| `book` | Books (id, name, star, language, file, size, deleted, date, format, id_inlib, archive, first_author_id, keys, id_lib) |
| `author` | Authors (id, name1/lastname, name2/firstname, name3/middlename, id_lib) |
| `seria` | Series (id, name, id_lib) |
| `genre` | Genres (id, name, keys - loaded from genre.tsv) |

#### Relationship Tables

| Table | Description |
|-------|-------------|
| `book_author` | Book-Author many-to-many (id_book, id_author, id_lib) |
| `book_sequence` | Book-Series relationship (id_book, id_sequence, num_in_sequence) |
| `book_genre` | Book-Genre many-to-many (id_book, id_genre, id_lib) |

#### Tagging System

| Table | Description |
|-------|-------------|
| `tag` | Tags/Labels (id, name, id_icon) |
| `icon` | Tag icons (id, dark_theme, light_theme - SVG data) |
| `book_tag` | Book-Tag relationship |
| `author_tag` | Author-Tag relationship |
| `seria_tag` | Series-Tag relationship |

### Data Structures

#### SBook (Book)
```go
type Book struct {
    ID            uint
    Name          string
    Annotation    string
    CoverImage    string
    Archive       string
    ISBN          string
    Date          time.Time
    Format        string
    File          string
    Keywords      string
    Genres        []uint16
    Authors       []uint
    Tags          []uint
    Series        map[uint]uint  // series_id -> number_in_series
    IDInLib       uint
    FirstAuthorID uint
    Size          uint
    Stars         uint8
    LanguageID    uint8
    Deleted       bool
}
```

#### SAuthor (Author)
```go
type Author struct {
    ID         uint
    FirstName  string
    LastName   string
    MiddleName string
    Tags       []uint
}
```

#### SSeries (Series)
```go
type Series struct {
    ID   uint
    Name string
    Tags []uint
}
```

#### SGenre (Genre)
```go
type Genre struct {
    ID           uint16
    Name         string
    Keys         []string
    ParentGenreID uint16
}
```

#### SLib (Library)
```go
type Library struct {
    ID           uint
    Name         string
    Path         string
    InpxPath     string
    Version      string
    FirstAuthor  bool
    WithoutDeleted bool
    Authors      map[uint]*Author
    Books        map[uint]*Book
    Series       map[uint]*Series
    Languages    []string
    EarliestDate time.Time
    Loaded       bool
}
```

### Genre System

The application uses a hierarchical genre system with 22 top-level categories and ~325 sub-genres. Genres are loaded from `genre.tsv`:

**Top-level categories:**
1. Фантастика (Fantasy/Sci-Fi)
2. Проза (Prose)
3. Наука, Образование (Science, Education)
4. Детективы и Триллеры (Detectives & Thrillers)
5. Документальная литература (Documentary)
6. Любовные романы (Romance)
7. Детское (Children's)
8. Домоводство (Home & Lifestyle)
9. Религия и духовность (Religion & Spirituality)
10. Приключения (Adventure)
11. Прочее (Other)
12. Юмор (Humor)
13. Поэзия (Poetry)
14. Справочная литература (Reference)
15. Техника (Technology)
16. Военное дело (Military)
17. Компьютеры и Интернет (Computers & Internet)
18. Драматургия (Drama)
19. Старинное (Antique Literature)
20. Деловая литература (Business)
21. Фольклор (Folklore)
22. Культура и искусство (Culture & Art)

### OPDS Server Routes

The original server supports three output formats:
- **HTML** - Web interface for browsers
- **OPDS** - Atom/XML feed for e-readers
- **OPDS2** - JSON-based modern OPDS format

#### Route Patterns

| Route Pattern | Description |
|---------------|-------------|
| `/{lib_id}` | Library root (HTML) |
| `/opds/{lib_id}` | Library root (OPDS) |
| `/opds2/{lib_id}` | Library root (OPDS2) |
| `/{lib_id}/authorsindex[/{prefix}]` | Authors index by letter |
| `/{lib_id}/author/{author_id}` | Author details |
| `/{lib_id}/authorbooks/{author_id}` | Author's books |
| `/{lib_id}/authorseries/{author_id}` | Author's series |
| `/{lib_id}/authorseries/{author_id}/{series_id}` | Books in author's series |
| `/{lib_id}/authorseriesless/{author_id}` | Author's books without series |
| `/{lib_id}/seriesindex[/{prefix}]` | Series index by letter |
| `/{lib_id}/seriesbooks/{series_id}` | Books in series |
| `/{lib_id}/genres[/{genre_id}]` | Genre browsing |
| `/{lib_id}/book/{book_id}/{format}` | Download book |
| `/{lib_id}/covers/{book_id}/cover.jpg` | Book cover image |
| `/{lib_id}/searchtitle` | Search by title |
| `/{lib_id}/searchauthor` | Search by author |
| `/{lib_id}/searchseries` | Search by series |
| `/assets/{file}` | Static assets (CSS, JS, images) |

### Import Process (INPX)

INPX files are ZIP archives containing `.inp` files with book metadata:

```
Field order in .inp files:
1. Authors (colon-separated)
2. Genre codes
3. Title
4. Series name
5. Number in series
6. Filename
7. File size
8. ID in library
9. Deleted flag
10. File format
11. Date added
12. Language
13. Rating (0-5)
14. Keywords
```

### Supported Book Formats

| Format | Read | Convert To |
|--------|------|------------|
| FB2 | ✓ | EPUB, MOBI, AZW3 |
| FB2.ZIP | ✓ | EPUB, MOBI, AZW3 |
| EPUB | ✓ | MOBI, AZW3 |
| MOBI | ✓ | - |
| AZW3 | ✓ | - |
| PDF | ✓ | - |
| DJVU | ✓ | - |

---

## Target Architecture

### Technology Stack (Go)

- **Language**: Go 1.21+
- **HTTP Router**: `chi` (lightweight, stdlib-compatible)
- **Database**: SQLite with `sqlx` (raw SQL for performance)
- **Template Engine**: `html/template` (standard library)
- **Frontend**: Custom CSS with CSS variables (no framework)
- **Archive Support**: `archive/zip` (standard library)
- **XML Parsing**: `encoding/xml` (standard library)

**Dependencies (minimal):**
- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/jmoiron/sqlx` - SQL extensions
- `github.com/mattn/go-sqlite3` - SQLite driver
- `gopkg.in/yaml.v3` - Config file parsing

### Project Structure

```
opds-server/
├── cmd/
│   └── opds-server/
│       └── main.go              # Application entry point, CLI
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration (YAML + env vars)
│   ├── db/
│   │   ├── db.go                # Database connection (sqlx)
│   │   ├── schema.sql           # Database schema
│   │   ├── models.go            # Data structures
│   │   ├── library.go           # Library queries
│   │   ├── book.go              # Book queries
│   │   ├── author.go            # Author queries
│   │   └── genre.go             # Genre queries
│   ├── importer/
│   │   ├── inpx.go              # INPX file parser
│   │   └── importer.go          # Import orchestration
│   ├── opds/
│   │   ├── opds.go              # OPDS Atom/XML generation
│   │   ├── opds2.go             # OPDS2 JSON generation
│   │   └── search.go            # OpenSearch support
│   ├── server/
│   │   ├── server.go            # HTTP server (chi router)
│   │   ├── routes.go            # Route definitions
│   │   ├── handlers_opds.go     # OPDS handlers
│   │   ├── handlers_web.go      # Web UI handlers
│   │   ├── handlers_api.go      # REST API handlers
│   │   └── middleware.go        # Auth, logging
│   └── web/
│       ├── templates.go         # Template loading
│       └── render.go            # Template rendering helpers
├── web/
│   ├── templates/
│   │   ├── layout.html          # Base layout
│   │   ├── index.html           # Home page
│   │   ├── authors.html         # Authors tab
│   │   ├── series.html          # Series tab
│   │   ├── genres.html          # Genres tab
│   │   ├── search.html          # Search tab
│   │   ├── book.html            # Book details
│   │   └── settings.html        # Settings page
│   └── static/
│       ├── css/
│       │   └── style.css        # Custom CSS
│       ├── js/
│       │   └── app.js           # Minimal JS (optional)
│       └── img/
│           └── logo.svg
├── genres.tsv                   # Genre definitions (embedded)
├── go.mod
├── go.sum
├── Makefile
├── Specification.md
└── README.md
```

---

## Data Models

### Go Struct Definitions

```go
// internal/db/models.go

type Library struct {
    ID              int64     `db:"id" json:"id"`
    Name            string    `db:"name" json:"name"`
    Path            string    `db:"path" json:"path"`
    InpxPath        string    `db:"inpx" json:"inpx_path"`
    Version         string    `db:"version" json:"version"`
    FirstAuthorOnly bool      `db:"first_author" json:"first_author_only"`
    WithoutDeleted  bool      `db:"without_deleted" json:"without_deleted"`
    CreatedAt       time.Time `db:"created_at" json:"created_at"`
    UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

type Book struct {
    ID            int64     `db:"id" json:"id"`
    LibraryID     int64     `db:"library_id" json:"library_id"`
    Title         string    `db:"title" json:"title"`
    Lang          string    `db:"lang" json:"lang"`
    File          string    `db:"file" json:"file"`
    Archive       string    `db:"archive" json:"archive"`
    Format        string    `db:"format" json:"format"`
    Size          int64     `db:"size" json:"size"`
    Rating        int       `db:"rating" json:"rating"`
    Deleted       bool      `db:"deleted" json:"deleted"`
    AddedAt       time.Time `db:"added_at" json:"added_at"`
    IDInLib       int64     `db:"id_in_lib" json:"id_in_lib"`
    FirstAuthorID int64     `db:"first_author_id" json:"first_author_id"`
    Keywords      string    `db:"keywords" json:"keywords"`
}

type Author struct {
    ID         int64  `db:"id" json:"id"`
    LibraryID  int64  `db:"library_id" json:"library_id"`
    LastName   string `db:"last_name" json:"last_name"`
    FirstName  string `db:"first_name" json:"first_name"`
    MiddleName string `db:"middle_name" json:"middle_name"`
}

// FullName returns formatted author name
func (a Author) FullName() string {
    name := a.LastName
    if a.FirstName != "" {
        name += " " + a.FirstName
    }
    if a.MiddleName != "" {
        name += " " + a.MiddleName
    }
    return name
}

type Series struct {
    ID        int64  `db:"id" json:"id"`
    LibraryID int64  `db:"library_id" json:"library_id"`
    Name      string `db:"name" json:"name"`
}

type Genre struct {
    ID       int    `db:"id" json:"id"`
    ParentID int    `db:"parent_id" json:"parent_id"`
    Name     string `db:"name" json:"name"`
    Code     string `db:"code" json:"code"` // genre code(s), comma-separated
}

// Join tables
type BookAuthor struct {
    BookID   int64 `db:"book_id"`
    AuthorID int64 `db:"author_id"`
}

type BookSeries struct {
    BookID   int64 `db:"book_id"`
    SeriesID int64 `db:"series_id"`
    SeqNum   int   `db:"seq_num"` // position in series
}

type BookGenre struct {
    BookID  int64 `db:"book_id"`
    GenreID int   `db:"genre_id"`
}
```

---

## API Endpoints

### REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/libraries` | List all libraries |
| POST | `/api/libraries` | Create new library |
| GET | `/api/libraries/{id}` | Get library details |
| PUT | `/api/libraries/{id}` | Update library |
| DELETE | `/api/libraries/{id}` | Delete library |
| POST | `/api/libraries/{id}/import` | Import from INPX |
| GET | `/api/libraries/{id}/books` | List books (with filters) |
| GET | `/api/libraries/{id}/authors` | List authors |
| GET | `/api/libraries/{id}/series` | List series |
| GET | `/api/libraries/{id}/genres` | List genres |
| GET | `/api/books/{id}` | Get book details |
| GET | `/api/books/{id}/download` | Download book file |
| GET | `/api/books/{id}/cover` | Get book cover |
| GET | `/api/authors/{id}` | Get author details |
| GET | `/api/authors/{id}/books` | Get author's books |
| GET | `/api/series/{id}` | Get series details |
| GET | `/api/series/{id}/books` | Get books in series |
| GET | `/api/search` | Global search |

### OPDS Endpoints

| Endpoint | Description |
|----------|-------------|
| `/opds/{lib_id}` | OPDS catalog root |
| `/opds/{lib_id}/new` | New books feed |
| `/opds/{lib_id}/authors` | Authors navigation |
| `/opds/{lib_id}/authors/{id}` | Author's books |
| `/opds/{lib_id}/series` | Series navigation |
| `/opds/{lib_id}/series/{id}` | Series books |
| `/opds/{lib_id}/genres` | Genres navigation |
| `/opds/{lib_id}/genres/{id}` | Genre books |
| `/opds/{lib_id}/search` | Search endpoint |

---

## Implementation Plan

### Priority Matrix
| Feature | Priority | Phase |
|---------|----------|-------|
| OPDS server | HIGH | 3 |
| INPX import | HIGH | 2 |
| Web browsing UI | MEDIUM | 4 |
| Search functionality | MEDIUM | 4 |
| Book download | LOW | 3 |
| Cover extraction | LOW | 5 |
| Format conversion | LOW | Future |

---

### Phase 1: Foundation
**Status: COMPLETED**

- [x] Initialize Go module and project structure
- [x] Set up SQLite database with sqlx
- [x] Implement database schema and migrations
- [x] Create configuration management (YAML + env vars)
- [x] Load genre data from embedded TSV
- [x] Basic CLI with flags (port, db path, config file, --restart)

**Deliverable:** Server starts, creates empty database, loads genres ✓

---

### Phase 2: Library Import (HIGH PRIORITY)
**Status: COMPLETED**

- [x] Implement INPX parser (ZIP with .inp files)
- [x] Parse .inp file format (author, title, series, genre, etc.)
- [x] Import books, authors, series to database
- [x] Handle genre code → genre ID mapping
- [x] Library CRUD operations (add, update, delete)
- [x] CLI command: `go run . import --inpx /path/to/file.inpx`

**Deliverable:** Can import INPX file and query data from database ✓

---

### Phase 3: OPDS Server (HIGH PRIORITY)
**Status: COMPLETED**

- [x] HTTP server setup with chi router
- [x] OPDS Atom/XML feed generation
  - [x] Root catalog
  - [x] Authors navigation (alphabetical index)
  - [x] Author's books
  - [x] Series navigation
  - [x] Series books
  - [x] Genres navigation
  - [x] Genre books
  - [x] OpenSearch description
- [ ] OPDS2 JSON feed generation (future)
- [x] Book file serving from ZIP archives
- [x] Basic authentication (optional)
- [x] Pagination support

**Deliverable:** E-reader can browse and download books via OPDS ✓

---

### Phase 4: Web UI (MEDIUM PRIORITY)
**Status: NOT STARTED**

- [ ] HTML templates with Go template engine
- [ ] Custom CSS (light/dark mode, responsive)
- [ ] Main layout (header, tabs, three-panel)
- [ ] Authors tab with alphabet filter
- [ ] Series tab
- [ ] Genres tab (hierarchical)
- [ ] Search tab with filters
- [ ] Book list table (sortable)
- [ ] Book details panel
- [ ] Settings page
- [ ] Library management UI

**Deliverable:** Full web interface for browsing library

---

### Phase 5: Enhancements (LOW PRIORITY)
**Status: PARTIAL**

- [ ] Cover image extraction from FB2/EPUB
- [ ] Cover caching
- [ ] Annotation extraction
- [x] Book file download with proper MIME types
- [x] OPDS search with case-insensitive Cyrillic support
- [ ] Tag management
- [x] Library delete CLI command (`go run . delete-library --id <id>`)
- [x] Library delete API endpoint (`DELETE /api/libraries/{id}`)

---

### Phase 6: User Authentication & Permissions
**Status: COMPLETED**

- [x] User database table (id, username, password_hash, role, created_at)
- [x] Session table for Web UI authentication
- [x] Roles: `admin` (full access) and `readonly` (browse/download only)
- [x] Admin-only actions:
  - Import library (CLI)
  - Delete library (API)
  - User management (API)
- [x] CLI commands always have admin permissions (no auth required)
- [x] Session management for Web UI (cookie-based)
- [x] Password hashing (bcrypt)
- [x] First-run setup endpoint (`POST /api/setup`)
- [x] OPDS Basic Auth support for e-readers
- [x] CLI command: `go run . create-user --username <user> --password <pass> [--role admin|readonly]`

**API Endpoints:**
- `GET /api/setup/check` - Check if setup is required
- `POST /api/setup` - Create initial admin user
- `POST /api/auth/login` - Login with username/password
- `POST /api/auth/logout` - Logout
- `GET /api/auth/me` - Get current user
- `GET /api/users` - List users (admin only)
- `POST /api/users` - Create user (admin only)
- `PUT /api/users/{id}/password` - Update password (admin only)
- `PUT /api/users/{id}/role` - Update role (admin only)
- `DELETE /api/users/{id}` - Delete user (admin only)

---

### Future Enhancements
- Format conversion (FB2 → EPUB, MOBI)
- Reading progress tracking
- Docker support
- Mobile app

---

## Test Dataset

A self-contained test library is included in `testdata/library/`:

```
testdata/library/
├── test-library.inpx       (2.2 KB) - INPX index with .inp files inside
├── fb2-000024-030559.zip   (1.8 MB) - 36 FB2 books
└── fb2-030560-060423.zip   (392 KB) - 10 FB2 books
```

**Total: 46 books, 2 archives**

Usage:
```bash
# Import test library
go run . import --inpx ./testdata/library/test-library.inpx \
    --name "Test Library" --path ./testdata/library

# Delete library
go run . delete-library --id 1
```

---

## Configuration

### Environment Variables

```bash
OPDS_SERVER_PORT=9988
OPDS_SERVER_HOST=0.0.0.0
OPDS_DATABASE_PATH=./data/library.db
OPDS_LIBRARY_PATH=./libraries
OPDS_CACHE_PATH=./cache
OPDS_AUTH_ENABLED=false
OPDS_AUTH_USER=admin
OPDS_AUTH_PASSWORD=
OPDS_LOG_LEVEL=info
```

### Config File (config.yaml)

```yaml
server:
  host: 0.0.0.0
  port: 9988

database:
  path: ./data/library.db

library:
  path: ./libraries
  cache_path: ./cache
  books_per_page: 50

auth:
  enabled: false
  user: admin
  password_hash: ""

opds:
  show_covers: true
  show_annotations: true
```

---

## Web UI Design

Based on the original desktop application, the web interface will replicate the key functionality with a responsive layout.

### Layout Structure

```
┌─────────────────────────────────────────────────────────────────────────┐
│  [Logo] opds-server    [Library Selector ▼]    [Settings ⚙]  [Language] │
├─────────────────────────────────────────────────────────────────────────┤
│  [Authors] [Series] [Genres] [Search]                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  Alphabet Filter: А Б В Г Д Е Ж З И Й К Л М Н О П Р С Т У Ф Х Ц Ч Ш Щ Ы │
│                   A B C D E F G H I J K L M N O P Q R S T U V W X Y Z   │
├────────────────┬────────────────────────────┬───────────────────────────┤
│                │                            │                           │
│  LEFT PANEL    │  CENTER PANEL              │  RIGHT PANEL              │
│  (Navigation)  │  (Book List)               │  (Book Details)           │
│                │                            │                           │
│  - Authors     │  ┌─────────────────────┐   │  ┌───────────────────┐    │
│    with count  │  │ Title    │#│Size│...│   │  │    [Cover Image]  │    │
│                │  ├─────────────────────┤   │  └───────────────────┘    │
│  - Series      │  │ ▼ Series Name       │   │  Title                    │
│    tree view   │  │   Book 1        │   │   │  Author: [link]           │
│                │  │   Book 2        │   │   │  Genre: [links]           │
│  - Genres      │  │ ▼ Another Series    │   │  Series: [link]           │
│    hierarchy   │  │   Book 3        │   │   │                           │
│                │  └─────────────────────┘   │  [Annotation text...]     │
│                │                            │                           │
│                │  [Pagination: < 1 2 3 >]   │  File: path/to/file       │
│                │                            │  Size: 1.2 MB             │
│                │                            │  Date: 21.11.2024         │
│                │                            │                           │
│                │                            │  [Download ▼] [fb2][epub] │
└────────────────┴────────────────────────────┴───────────────────────────┘
```

### Main Views

#### 1. Authors Tab
- **Left panel**: Alphabetical list of authors with book counts
- **Center panel**: Selected author's books grouped by series (tree view)
- **Right panel**: Selected book details with cover

#### 2. Series Tab
- **Left panel**: Alphabetical list of series with book counts
- **Center panel**: Books in selected series
- **Right panel**: Book details

#### 3. Genres Tab
- **Left panel**: Hierarchical genre tree (parent → children)
- **Center panel**: Books in selected genre
- **Right panel**: Book details

#### 4. Search Tab
- **Left panel**: Search filters
  - Title
  - Author
  - Genre (dropdown)
  - Series
  - Filename
  - Date range
  - Language (dropdown)
  - Minimum rating (stars)
  - Max results
- **Center panel**: Search results
- **Right panel**: Book details

### Book List Table Columns
| Column | Description |
|--------|-------------|
| Title (Название) | Book title, sortable |
| # (№) | Number in series |
| Size (Размер) | File size |
| Rating (Оценка) | Star rating (1-5) |
| Added (Добавлена) | Date added |
| Genre (Жанр) | Primary genre |
| Language (Язык) | Book language |

### Book Details Panel
- Cover image (if available)
- Title
- Author (clickable link)
- Genre (clickable links)
- Series with number (clickable link)
- Annotation/description
- File information:
  - Path to file
  - Filename
  - File size (archive size)
  - Creation date
- Download buttons for available formats

### Settings Page
Accessible via gear icon, includes:

**Server Settings:**
- Port number
- Base URL
- Books per page
- Show covers toggle
- Show annotations toggle
- Password protection (user/password)

**General Settings:**
- Show deleted books
- Use tags for filtering
- Interface language
- Database path

**Library Management:**
- Add/Edit/Delete libraries
- Library name, path, INPX file
- Update mode (add new / rebuild)
- Import options

### Mobile Responsive Design
On screens < 900px:
- Single column layout
- Collapsible panels
- Bottom navigation tabs
- Touch-friendly controls

### CSS Approach
- **Custom CSS** (no framework)
- Light/dark mode via `prefers-color-scheme`
- CSS variables for theming
- Minimal, clean design (~300-400 lines)

---

## Notes

### Differences from Original

1. **No GUI** - Web-only interface instead of Qt desktop application
2. **No email sending** - Removed email export feature (can be added later)
3. **Simplified conversion** - May rely on external tools (calibre) for format conversion
4. **No system tray** - Server runs as a daemon/service

### Future Enhancements

1. User accounts and permissions
2. Reading progress tracking
3. Book recommendations
4. Integration with external metadata sources (Google Books, OpenLibrary)
5. Mobile-responsive web interface
6. WebSocket for real-time updates during import
