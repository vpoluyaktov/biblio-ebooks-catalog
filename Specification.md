# Biblio OPDS Server

> Part of the [BiblioHub](https://github.com/vpoluyaktov/BiblioHub) application suite

A Go-based web server for managing e-book libraries with OPDS catalog support for e-readers.

## Overview

Biblio OPDS Server provides:
- **Web-based UI** for browsing and managing e-book libraries
- **OPDS catalog server** for e-reader compatibility
- **REST API** for programmatic access
- **Multi-library support** with SQLite database
- **INPX import** for library catalogs
- **User authentication** with admin and readonly roles

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Web Browser / E-Reader                    │
└─────────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
   │   Web UI    │     │  OPDS Feed  │     │  REST API   │
   │   (HTML)    │     │  (Atom/XML) │     │   (JSON)    │
   └─────────────┘     └─────────────┘     └─────────────┘
          │                   │                   │
          └───────────────────┼───────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      OPDS Server (Go)                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────────┐  │
│  │ chi HTTP │  │  SQLite  │  │ Importer │  │  Book Files    │  │
│  │  Router  │  │    DB    │  │  (INPX)  │  │  (ZIP/FB2)     │  │
│  └──────────┘  └──────────┘  └──────────┘  └────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
biblio-opds-server/
├── main.go                      # Application entry point
├── Specification.md             # This file
├── internal/
│   ├── config/config.go         # Configuration management
│   ├── db/
│   │   ├── db.go                # Database connection
│   │   ├── models.go            # Data structures
│   │   ├── library.go           # Library queries
│   │   ├── book.go              # Book queries
│   │   └── author.go            # Author queries
│   ├── importer/inpx.go         # INPX file parser
│   ├── bookfile/                # Book file handling, covers
│   └── server/
│       ├── server.go            # HTTP server
│       ├── handlers_opds.go     # OPDS handlers
│       ├── handlers_web.go      # Web UI handlers
│       └── handlers_api.go      # REST API handlers
├── web/
│   ├── templates/               # HTML templates
│   └── static/                  # CSS, JS, images
└── testdata/library/            # Test dataset
```

## REST API

### Library Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/libraries` | List all libraries |
| `POST` | `/api/libraries` | Create new library |
| `GET` | `/api/libraries/{id}` | Get library details |
| `DELETE` | `/api/libraries/{id}` | Delete library |
| `GET` | `/api/libraries/{id}/books` | List books (with filters) |
| `GET` | `/api/libraries/{id}/authors` | List authors |
| `GET` | `/api/libraries/{id}/series` | List series |

### Book Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/books/{id}` | Get book details |
| `GET` | `/api/books/{id}/download` | Download book file |
| `GET` | `/api/books/{id}/cover` | Get book cover |

### Authentication Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/setup/check` | Check if setup required |
| `POST` | `/api/setup` | Create initial admin |
| `POST` | `/api/auth/login` | Login |
| `POST` | `/api/auth/logout` | Logout |
| `GET` | `/api/users` | List users (admin) |
| `POST` | `/api/users` | Create user (admin) |

## OPDS Endpoints

| Endpoint | Description |
|----------|-------------|
| `/opds/{lib_id}` | OPDS catalog root |
| `/opds/{lib_id}/new` | New books feed |
| `/opds/{lib_id}/authors` | Authors navigation |
| `/opds/{lib_id}/authors/{id}` | Author's books |
| `/opds/{lib_id}/series` | Series navigation |
| `/opds/{lib_id}/series/{id}` | Series books |
| `/opds/{lib_id}/genres` | Genres navigation |
| `/opds/{lib_id}/search` | Search endpoint |

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPDS_SERVER_HOST` | Server host | `0.0.0.0` |
| `OPDS_SERVER_PORT` | Server port | `9903` |
| `OPDS_DATABASE_PATH` | SQLite database path | `./data/library.db` |
| `OPDS_LIBRARY_PATH` | Book files directory | `./libraries` |
| `OPDS_LOG_LEVEL` | Logging level | `info` |

### Command Line

```bash
# Import library from INPX
go run . import --inpx /path/to/file.inpx --name "My Library" --path /path/to/books

# Delete library
go run . delete-library --id 1

# Create user
go run . create-user --username admin --password secret --role admin

# Start server
go run . --port 9903
```

## Web UI Features

### Desktop Experience
- **Three-panel layout**: Navigation, book list, book details
- **Tabs**: Authors, Series, Genres, Search
- **Virtual scrolling** for large lists
- **Keyboard navigation** in book list (Arrow keys, Page Up/Down, Home/End, Enter)
- **Resizable panels** with drag handles
- **Sortable columns** in book table
- **Light/dark mode** support

### Mobile-Friendly Design (Navigation-Based Architecture)
- **Separate mobile UI**: Completely independent mobile interface (≤768px) with navigation-based screens
- **Home screen menu**: Central hub with options for Authors, Series, Genres, Advanced Search, and Configuration
- **Screen-based navigation**: Each section is a full-screen view with back button navigation
- **Navigation flow**:
  - Home → Authors → Books → Book Detail
  - Home → Series → Books → Book Detail
  - Home → Genres → Books → Book Detail
  - Home → Advanced Search → Results → Book Detail
  - Home → Configuration (library selection, admin, logout)
- **Touch-optimized UI**: 44px minimum touch targets, large tap areas, smooth transitions
- **Virtual scrolling**: Efficient rendering of large lists (authors, series, books)
- **Filter functionality**: Real-time filtering on all list screens
- **Global search**: Quick search from home screen
- **Book details**: Full-screen view with cover image, metadata, and download button
- **Theme support**: Dark/light mode toggle available on all screens
- **Safe area support**: Respects device notches and safe areas
- **History management**: Back button navigation with screen history stack
- **Responsive images**: Optimized book cover loading and display

## Supported Book Formats

| Format | Read | Cover | Annotation |
|--------|------|-------|------------|
| FB2 | ✅ | ✅ | ✅ |
| FB2.ZIP | ✅ | ✅ | ✅ |
| EPUB | ✅ | ⏳ | ⏳ |
| MOBI | ✅ | ❌ | ❌ |
| PDF | ✅ | ❌ | ❌ |

## Docker Deployment

Part of BiblioHub Docker Swarm stack:

```yaml
opds-server:
  image: vpoluyaktov/bibliohub-opds-server:dev-latest
  ports:
    - "9903:9903"
  environment:
    - OPDS_SERVER_HOST=0.0.0.0
    - OPDS_SERVER_PORT=9903
  volumes:
    - ./data/opds/db:/db
    - ./data/opds/books:/books
```

## Development Status

**Current State**: Fully operational

### Completed ✅

- OPDS Atom/XML feed generation
- Web UI with three-panel layout
- INPX library import
- Book file serving from ZIP archives
- Cover image extraction (FB2)
- Placeholder cover generation
- User authentication (admin/readonly)
- Virtual scrolling for large libraries
- Search functionality
- SQLite with ICU support for Cyrillic
- **Mobile-native UI** (2026-01-25)
  - Complete navigation-based mobile interface (≤768px)
  - Separate mobile.js and mobile.css modules
  - Home screen with menu-driven navigation
  - Screen-based architecture with history management
  - Touch-optimized with 44px minimum touch targets
  - Full-screen views for all sections
  - Desktop UI completely preserved and unchanged

### In Progress 🚧

- **Universal Book Import (EPUB + FB2)** (2026-01-26)
  - Feature branch: `feature/EPUB_Import`
  - Goal: Import EPUB and FB2 files with or without INPX index file

#### Problem Statement

Currently, library import relies on INPX files (index files commonly used for FB2 libraries). However:
1. EPUB libraries typically don't have INPX files
2. Users may have mixed FB2/EPUB collections without an index
3. Manual metadata extraction from files is slow but necessary as fallback

#### Solution Overview

Implement a **three-mode system**:
- **INPX Import (Fast)**: Use existing INPX file if provided (current behavior)
- **Scan Import (Slow)**: Direct filesystem scan with metadata extraction from each book file, import to database
- **Reindex Mode**: Generate INPX file from database for an existing library

#### Supported Formats

| Format | Metadata | Cover | Annotation |
|--------|----------|-------|------------|
| FB2 | ✅ | ✅ | ✅ |
| FB2.ZIP | ✅ | ✅ | ✅ |
| EPUB | ✅ | ✅ | ✅ |

#### Import Modes

**1. INPX Import (Fast)**
- User provides path to INPX file
- Parse INPX for metadata (existing implementation)
- Fast: metadata already indexed

**2. Directory Scan Import (Slow)**
- User provides path to book directory (no INPX)
- Recursively scan for `.fb2`, `.fb2.zip`, `.epub` files
- Parse each file for metadata (matching INPX fields):
  - Title, Author(s), Series, Series Number
  - Genre codes, Language
  - File size, Format
- Import metadata directly to database
- Note: Cover images are extracted on-demand when requested, not during import
- Show progress indicator (file count, percentage)

**3. Reindex Mode**
- User selects existing library from database
- User specifies output INPX file path and name
- Export all library metadata from database to INPX format
- Overwrites existing INPX file if it exists
- Fast operation (reads from database, not files)
- Useful for:
  - Creating INPX for libraries imported via scan mode
  - Regenerating INPX after manual database edits
  - Sharing library index without book files

#### Web UI Changes

**Import Screen:**
- Add warning when importing without INPX:
  > "No index file provided. Import will scan all book files for metadata. This may take significantly longer than importing with an .inpx file."
- Show estimated time based on file count
- Progress bar during scan import
- Option to cancel long-running imports
- **"Recreate library" checkbox** - When checked and library with same path exists:
  - Show confirmation warning: "This will delete all existing books, authors, and series for this library. This action cannot be undone."
  - If confirmed, delete existing library before import

**Reindex Screen:**
- Library dropdown to select existing library
- File path input for output INPX file location
- File name input (defaults to `library_name.inpx`)
- "Overwrite if exists" checkbox
- Progress indicator during INPX generation
- Success message with file location

#### CLI Changes

```bash
# Import with INPX (existing)
./biblio-opds-server import --inpx /path/to/file.inpx --name "My Library" --path /path/to/books

# Import by scanning directory (new)
./biblio-opds-server import --scan --name "My Library" --path /path/to/books

# Recreate library (delete existing and reimport)
./biblio-opds-server import --scan --recreate --name "My Library" --path /path/to/books

# Generate INPX from existing library (new)
./biblio-opds-server reindex --library-id 1 --output /path/to/output.inpx
./biblio-opds-server reindex --library-name "My Library" --output /path/to/output.inpx
```

#### Recreate Mode Behavior

When `--recreate` is used with an existing library (matched by path):
1. **Delete existing library data** - All books, authors, series for that library are removed (CASCADE delete)
2. **Rescan directory** - Fresh scan of all book files
3. **Reimport to database** - Insert all scanned books as new records

Note: This is a destructive operation. The library ID may change. Any user customizations (ratings, etc.) will be lost.

#### Reindex Mode Behavior

When `reindex` command is used:
1. **Query database** - Fetch all books, authors, series for specified library
2. **Format as INPX** - Convert database records to INPX format (semicolon-delimited)
3. **Write INPX file** - Create ZIP archive with `.inp` file(s)
4. **Overwrite if exists** - Replace existing file at output path

Note: This is a non-destructive operation. Database remains unchanged.

#### Implementation Steps

- [x] Create test data fetcher script (`scripts/fetch_gutenberg_epubs.py`)
- [x] Create test EPUB library with varied directory structures
- [x] **Phase 1: EPUB Metadata Parser**
  - [x] Create `internal/bookfile/epub.go` for EPUB parsing
  - [x] Extract metadata from `META-INF/container.xml` → OPF file
  - [x] Parse OPF for: title, creator, language, series (calibre metadata), genre
  - [x] Unit tests for EPUB parser
  - [x] Add EPUB cover extraction function (for on-demand use, not import)
- [x] **Phase 2: Directory Scanner**
  - [x] Create `internal/importer/scanner.go` for filesystem scanning
  - [x] Recursive directory walk for supported extensions
  - [x] Parallel file processing with worker pool
  - [x] Progress reporting callback
  - [x] Handle various directory structures (flat, Author/Book, Author/Series/Book, Genre/Book)
- [x] **Phase 3: INPX Generator (Reindex Mode)**
  - [x] Create `internal/importer/inpx_writer.go`
  - [x] Query database for library books, authors, series
  - [x] Format records as INPX (semicolon-delimited)
  - [x] Write ZIP archive with `.inp` file(s)
  - [x] Support both library ID and library name lookup
- [x] **Phase 4: CLI Integration**
  - [x] Add `scan` command for directory scanning import
  - [x] Add `--recreate` flag to delete and reimport existing library
  - [x] Add `reindex` command with `--library-id`, `--library-name`, `--output` flags
  - [x] Progress output during scan and reindex
- [x] **Phase 5: Web UI Integration**
  - [x] API endpoints for scan import (`POST /api/libraries/scan`)
  - [x] API endpoints for reindex (`POST /api/libraries/reindex`)
  - [x] Support for recreate flag in scan import
  - [x] Validation of paths and parameters
  - [x] Admin-only access control
  - [ ] Frontend UI components (optional - CLI is fully functional)
- [x] **Phase 6: Testing & Documentation**
  - [x] Integration tests with test EPUB library (162 EPUBs tested)
  - [x] Update README with new import options
  - [x] CLI usage examples for all three modes
  - [x] API endpoint documentation

#### Technical Notes

**EPUB Structure:**
```
book.epub (ZIP archive)
├── META-INF/
│   └── container.xml      # Points to OPF file location
├── OEBPS/ (or similar)
│   ├── content.opf        # Metadata (Dublin Core + calibre extensions)
│   ├── toc.ncx            # Table of contents
│   ├── cover.jpg          # Cover image (referenced in OPF)
│   └── *.xhtml            # Content files
```

**INPX Structure:**
- ZIP archive containing `.inp` files
- Each `.inp` is semicolon-delimited text with book metadata
- Fields: Author;Genre;Title;Series;SeriesNum;File;Size;LibId;Deleted;Ext;Date;Language;Keywords;Annotation

**Performance Considerations:**
- Use worker pool (e.g., 4-8 workers) for parallel file parsing
- Stream large files instead of loading entirely into memory
- Cache extracted covers to avoid re-extraction
- Batch database inserts (100-500 records per transaction)

**Code Reuse:**
- Reuse ebook parsers from `biblio-audiobook-builder-tts/internal/parser/`:
  - `epub.go` - EPUB metadata and content extraction
  - `fb2.go` - FB2 metadata and content extraction
  - `parser.go` - Common parser interface
- These parsers are well-tested with comprehensive test suites (`epub_test.go`, `fb2_test.go`)

### Future Enhancements

- OPDS2 JSON feed
- Format conversion (FB2 → EPUB)
- Reading progress tracking

## Dependencies

### Go Modules

- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/mattn/go-sqlite3` - SQLite driver (with ICU)
- `github.com/fogleman/gg` - Cover image generation

### Build Requirements

- Go 1.21+
- ICU libraries (`libicu-dev`) for Cyrillic case conversion
- Build with `-tags "icu"` flag

---

*Last updated: 2026-01-26*
