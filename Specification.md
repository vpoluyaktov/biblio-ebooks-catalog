# Biblio Catalog

> Part of the [BiblioHub](https://github.com/vpoluyaktov/biblio-hub) application suite

A Go-based web server for managing e-book libraries with OPDS catalog support for e-readers.

## Overview

Biblio Catalog provides:
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
│  │ net/http │  │  SQLite  │  │ Importer │  │  Book Files    │  │
│  │ ServeMux │  │    DB    │  │  (INPX)  │  │  (ZIP/FB2)     │  │
│  └──────────┘  └──────────┘  └──────────┘  └────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
biblio-ebooks-catalog/
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

OPDS feeds are served at `{BASE_PATH}/opds/{lib_id}/...` where `BASE_PATH` is configurable (e.g., `/catalog`).

| Endpoint | Description |
|----------|-------------|
| `{BASE_PATH}/opds/{lib_id}` | OPDS catalog root |
| `{BASE_PATH}/opds/{lib_id}/authors` | Authors navigation |
| `{BASE_PATH}/opds/{lib_id}/authors/{letter}` | Authors by letter |
| `{BASE_PATH}/opds/{lib_id}/author/{id}` | Author's books |
| `{BASE_PATH}/opds/{lib_id}/series` | Series navigation |
| `{BASE_PATH}/opds/{lib_id}/series/{id}` | Series books |
| `{BASE_PATH}/opds/{lib_id}/genres` | Genres navigation |
| `{BASE_PATH}/opds/{lib_id}/genres/{id}` | Genre books |
| `{BASE_PATH}/opds/{lib_id}/book/{id}/{format}` | Download book |
| `{BASE_PATH}/opds/{lib_id}/covers/{id}/cover.jpg` | Book cover image |
| `{BASE_PATH}/opds/{lib_id}/annotation/{id}` | Book annotation |
| `{BASE_PATH}/opds/{lib_id}/search` | Search endpoint |
| `{BASE_PATH}/opds/{lib_id}/opensearch.xml` | OpenSearch descriptor |

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPDS_SERVER_HOST` | Server host | `0.0.0.0` |
| `OPDS_SERVER_PORT` | Server port | `80` |
| `OPDS_BASE_PATH` | Base URL path for path-based routing | `/catalog` |
| `AUTH_MODE` | Authentication mode (`internal` or `oidc`) | `oidc` |
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

Part of BiblioHub Docker Swarm stack. Access via `http://localhost:9900/catalog/`

```yaml
biblio-catalog:
  image: vpoluyaktov/bibliohub-catalog:dev-latest
  environment:
    - OPDS_SERVER_HOST=0.0.0.0
    - OPDS_SERVER_PORT=80
    - OPDS_BASE_PATH=/catalog
    - AUTH_MODE=${AUTH_MODE:-oidc}
    - OIDC_URL=http://${BIBLIO_HUB_HOSTNAME:-localhost}:${BIBLIO_HUB_PORT:-9900}/auth
    - OIDC_REALM=biblio
    - OIDC_CLIENT_ID=biblio-catalog
  volumes:
    - ./data/opds/db:/db
    - ${EBOOKS_PATH}:/books:ro
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
- **Path-Based Routing Refactoring** (2026-01-27)
  - Removed chi router dependency completely
  - Migrated to standard `net/http.ServeMux` with manual path parsing
  - All handlers accept IDs as function parameters (no context extraction)
  - Base path support for deployment under sub-paths (e.g., `/catalog`)
  - Fixed OPDS feed URL generation to use configured base path
  - Fixed JavaScript cover URL generation to use `APP_BASE_PATH`
  - Fixed API route ordering to prevent `/libraries/import` conflict
  - Consistent routing approach across API and OPDS endpoints

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

**2. Directory Scan Import (Slow) - Streaming Import Flow**

The streaming import flow ensures:
- **Immediate progress feedback** - UI shows accurate progress from the start
- **Partial import preservation** - Canceled imports save already-processed books
- **Responsive cancellation** - Cancel button stops import within seconds
- **Memory efficiency** - Books imported as parsed, not accumulated in memory

**Flow Steps:**

1. **File Discovery Phase** (Fast - seconds)
   - Recursively scan directory for book files
   - Collect file paths and determine file types:
     - `.fb2` → FB2 type
     - `.epub` → EPUB type  
     - `.zip` → ZIP type (requires further inspection)
   - For ZIP files: Quick list of contents to identify books inside
     - Store each book as "InZip" type with:
       - Parent ZIP file path
       - File name inside ZIP
       - Position/offset in ZIP (for efficient extraction)
   - **Result**: Complete file list with accurate total book count for progress bar

2. **Streaming Parse & Import Phase** (Slow - minutes)
   - Process files one-by-one in a loop
   - **Before each book**:
     - Check cancellation flag → exit if canceled
     - Update progress bar (current/total)
     - Update import statistics (imported/skipped counts)
   
   - **For single FB2/EPUB files**:
     - Parse metadata using appropriate parser (FB2Parser or EPUBParser)
     - Import to database immediately
     - Commit transaction every 100 books
   
   - **For InZip files**:
     - Extract file from ZIP on-the-fly (no temp files)
     - Parse metadata from bytes using `ParseFromReader()`
     - Import to database immediately
     - Commit transaction every 100 books
   
   - **After each book**:
     - Increment progress counter
     - Send progress update to UI via SSE
     - Check if batch size (100) reached → commit transaction

3. **Completion/Cancellation**
   - On completion: Final commit, show success message with stats
   - On cancellation: Final commit of partial batch, show canceled message with stats
   - Library exists with all successfully imported books (even if canceled)

**Key Design Principles:**
- **No bulk collection**: Don't collect all parsed books in memory before importing
- **Frequent commits**: Commit every 100 books to preserve progress
- **Frequent cancellation checks**: Check before each book parse (not just between batches)
- **Accurate progress**: Total count known from file discovery phase
- **On-the-fly ZIP extraction**: Extract and parse ZIP contents without temp files

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
./biblio-catalog import --inpx /path/to/file.inpx --name "My Library" --path /path/to/books

# Import by scanning directory (new)
./biblio-catalog import --scan --name "My Library" --path /path/to/books

# Recreate library (delete existing and reimport)
./biblio-catalog import --scan --recreate --name "My Library" --path /path/to/books

# Generate INPX from existing library (new)
./biblio-catalog reindex --library-id 1 --output /path/to/output.inpx
./biblio-catalog reindex --library-name "My Library" --output /path/to/output.inpx
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
- [x] **Phase 7: FB2 Parser Robustness** (2026-01-27)
  - [x] Extract FB2 parser from `epub.go` to separate `fb2.go` file
  - [x] Fix series number parsing to handle non-numeric values
    - Parse year-month formats (`"1996 02"` → 1996)
    - Parse underscore/dash formats (`"09_2"` → 9)
    - Handle corrupted data (`"« name=»Рассказы"` → 1)
    - Default to 1 for unparseable values
  - [x] Add XML sanitization for malformed FB2 files
    - Fix invalid UTF-8 sequences (Windows-1251 auto-detection)
    - Remove illegal XML control characters (U+0001-U+001F except tab/newline/CR)
    - Escape unescaped ampersands
    - Fix malformed tags (tags starting with numbers, ellipsis, etc.)
  - [x] Comprehensive unit tests covering all error categories
    - Series number parsing (20 test cases)
    - UTF-8 fixing (3 test cases)
    - Illegal character removal (7 test cases)
    - Ampersand escaping (9 test cases)
    - Malformed tag fixing (6 test cases)
    - Full FB2 parsing with malformed XML (4 test cases)
  - [x] Set `decoder.Strict = false` for lenient XML parsing
  - [x] Fallback to original data if sanitization fails
- [x] **Phase 8: Parser Interface Refactoring** (2026-01-27)
  - [x] Create unified `Parser` interface in `internal/parser/parser.go`
    - `Parse(filePath string) (*Metadata, error)` - parse from file
    - `ParseFromBytes(data []byte) (*Metadata, error)` - parse from bytes
    - `ParseFromReader(reader io.Reader) (*Metadata, error)` - parse from reader
    - `Format() string` - return format identifier
  - [x] Implement `EPUBParser` and `FB2Parser` types conforming to interface
  - [x] Create `Registry` for managing parsers by format
  - [x] Add convenience functions: `Parse(format, path)` and `ParseFromBytes(format, data)`
  - [x] Unified `Metadata` struct replacing format-specific metadata types
    - Title, Authors ([]string), Language, Description
    - Genres, Series, SeriesIndex
    - CoverData, CoverType
  - [x] Update all consumers to use parser interface:
    - `internal/importer/scanner.go` - uses `parser.Parse()` for scanning
    - `internal/importer/scanner_zip.go` - uses `parser.ParseFromBytes()` for ZIP archives
    - `internal/server/handlers_opds.go` - uses `parser.Parse()` for cover/annotation extraction
  - [x] Remove backward compatibility code - single clean interface throughout
  - [x] All 49 parser tests passing

#### Error Analysis from Production Import

During FB2 library import (3,006 errors analyzed):
- **Series number parsing errors (~400)**: Fixed by lenient string parsing
- **Invalid UTF-8 (~29)**: Fixed by charset auto-detection
- **Illegal XML characters (~100)**: Fixed by character filtering
- **Unescaped ampersands (~76)**: Fixed by entity escaping
- **Malformed XML tags (~2,400)**: Partially fixed, some files may remain unparseable due to severe corruption

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

## Authentication

Biblio Catalog supports two authentication modes:

### Internal Mode (`AUTH_MODE=internal`)
- Standalone deployment with local SQLite user database
- Session-based authentication with cookies
- HTTP Basic Auth support for e-readers (OPDS clients)
- User management via web UI

### OIDC Mode (`AUTH_MODE=oidc`)
- Integration with Keycloak for centralized SSO
- OAuth2 Authorization Code flow for web UI
- HTTP Basic Auth for e-readers (validated via Keycloak ROPC)
- User management via Keycloak Admin Console

## Dependencies

### Go Modules

- `github.com/mattn/go-sqlite3` - SQLite driver (with ICU)
- `github.com/fogleman/gg` - Cover image generation
- `github.com/coreos/go-oidc/v3/oidc` - OIDC client (for Keycloak)
- `golang.org/x/oauth2` - OAuth2 client
- Standard library `net/http` - HTTP server and routing

### Build Requirements

- Go 1.21+
- ICU libraries (`libicu-dev`) for Cyrillic case conversion
- Build with `-tags "icu"` flag

---

## Recent Changes

### fix/oidc-login-flash (2026-01-29)
- **Issue**: When accessing the catalog in OIDC mode, the internal login screen would briefly flash before redirecting to Keycloak
- **Root Cause**: The `router()` function was redirecting to `#login` before `checkAuth()` could complete the OIDC redirect
- **Fix**: Added `oidcRedirectPending` flag to track when OIDC redirect is in progress, preventing the router from showing the internal login screen and keeping the loading spinner visible during redirect

- **Issue**: Logout button in OIDC mode showed internal login screen instead of logging out via Keycloak
- **Root Cause**: The `logout()` function only called internal logout endpoint and redirected to `#login`
- **Fix**: Updated `logout()` to check auth mode and call OIDC logout endpoint, then redirect to Keycloak logout URL. Also fixed missing scheme in redirect URL construction in backend.

---

*Last updated: 2026-01-29*
