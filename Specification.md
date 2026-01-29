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
│  │ net/http │  │  SQLite  │  │ Importer │  │  Book Files    │  │
│  │ ServeMux │  │    DB    │  │  (INPX)  │  │  (ZIP/FB2)     │  │
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
| `OPDS_SERVER_PORT` | Server port | `9903` |
| `OPDS_BASE_PATH` | Base URL path for deployment | `/catalog` |
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

## Dependencies

### Go Modules

- `github.com/mattn/go-sqlite3` - SQLite driver (with ICU)
- `github.com/fogleman/gg` - Cover image generation
- Standard library `net/http` - HTTP server and routing

### Build Requirements

- Go 1.21+
- ICU libraries (`libicu-dev`) for Cyrillic case conversion
- Build with `-tags "icu"` flag

---

## Recent Changes

### Path-Based Routing Implementation (2026-01-28)

**Overview**: Implemented support for running the OPDS server on a sub-path (e.g., `/opds`) as part of the BiblioHub unified deployment architecture.

**Architecture Approach**:
- Backend routes registered **WITH** base path prefix (e.g., `/opds/api/libraries`)
- Nginx gateway preserves full path when forwarding (no trailing slash in `proxy_pass`)
- Frontend JavaScript uses `apiUrl()` helper to prepend base path to all API calls
- Each service isolated in its own namespace, preventing route conflicts

**Backend Changes**:
- Added `BASE_PATH` environment variable support in `config.go` (default: `/`)
- Routes registered with base path prefix in `server.go`
- Path parsing in handlers uses `strings.Index()` to find prefix position
- Fixed `handleOPDSSource()` to correctly extract source ID with base path

**Frontend Changes**:
- Added `window.APP_BASE_PATH` injection in HTML template
- Created `apiUrl()` helper function in `app.js` to prefix all API calls
- Fixed mobile UI (`mobile.js`): 4 OPDS feed URLs now use `App.apiUrl()`
  - Author books URL
  - Series books URL  
  - Genre books URL
  - Search URL
- Fixed user management (`app.js`): 2 endpoints now use `this.apiUrl()`
  - Change password endpoint
  - Change role endpoint

**Configuration**:
```yaml
# stack.yaml
opds-server:
  environment:
    - BASE_PATH=/opds
```

**Impact**: 
- Resolves "no books found" issue in mobile UI when selecting authors/series
- Fixes user management operations (password/role changes)
- Enables deployment behind reverse proxy with path-based routing
- All API calls now work correctly with base path prefix

---

### refactor/centralize-basepath-handling - Centralized Base Path Handling (2026-01-28)

**Problem**: Base path handling was scattered throughout the codebase with manual string concatenation (`s.config.Server.BasePath + "/path"`), making the code harder to maintain and more prone to errors.

**Solution**:
- Created `Server.apiURL()` helper method to centralize base path handling in the backend
- Updated all OPDS handlers to use the helper method instead of manual concatenation
- Frontend already had `App.apiUrl()` helper that works consistently
- Reduced code duplication and improved maintainability

**Files Changed**:
- `internal/server/server.go` - Added `apiURL()` helper method
- `internal/server/handlers_opds.go` - Replaced 9 instances of manual `basePath` concatenation with `apiURL()` calls

**Impact**:
- Centralized base path logic in a single helper method
- Easier to maintain and modify base path handling in the future
- Consistent pattern between frontend and backend
- Reduced code duplication across OPDS handlers

---

### Dual Authentication Mode Support (2026-01-28)

**Overview**: Implemented support for two authentication modes to enable both standalone deployment (internal auth) and BiblioHub swarm deployment (Keycloak SSO).

**Authentication Modes**:

1. **Internal Mode** (`AUTH_MODE=internal`)
   - Standalone deployment with local SQLite user database
   - Session-based authentication with cookies
   - HTTP Basic Auth support for e-readers (OPDS clients)
   - User management via web UI
   - Default mode for standalone Docker containers

2. **Keycloak Mode** (`AUTH_MODE=keycloak`)
   - Integration with Keycloak for centralized SSO
   - OAuth2 Authorization Code flow
   - Token-based authentication (ID tokens, access tokens)
   - User management via Keycloak Admin Console
   - Default mode for BiblioHub swarm deployment

**Architecture**:

```
┌─────────────────────────────────────────────────────────┐
│                    Auth Manager                          │
│  (Factory pattern - switches based on AUTH_MODE)        │
└─────────────────────────────────────────────────────────┘
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
┌──────────────────┐    ┌──────────────────┐
│  Internal Auth   │    │  Keycloak Auth   │
│  - SQLite DB     │    │  - OIDC Provider │
│  - bcrypt hash   │    │  - OAuth2 flow   │
│  - Sessions      │    │  - JWT tokens    │
│  - Basic Auth    │    │  - Role mapping  │
└──────────────────┘    └──────────────────┘
```

**Implementation Details**:

- **Config**: Added `AUTH_MODE`, `KeycloakConfig` to `internal/config/config.go`
- **Auth Manager**: Created `internal/auth/manager.go` - factory pattern for mode switching
- **Keycloak Provider**: Created `internal/auth/keycloak.go` - OIDC/OAuth2 implementation
- **Handlers**: 
  - New: `internal/server/handlers_keycloak.go` - Keycloak login/callback/logout
  - Updated: `internal/server/handlers_auth.go` - Mode-aware internal auth handlers
- **Server**: Updated `internal/server/server.go` to use auth manager

**Environment Variables**:

```bash
# Authentication mode
AUTH_MODE=keycloak  # or 'internal'

# Keycloak configuration (required when AUTH_MODE=keycloak)
KEYCLOAK_URL=http://localhost:9900/auth
KEYCLOAK_REALM=biblio
KEYCLOAK_CLIENT_ID=opds-server
KEYCLOAK_CLIENT_SECRET=your-secret-here
KEYCLOAK_REDIRECT_URL=http://localhost:9900/catalog/api/auth/keycloak/callback
```

**API Endpoints**:

Internal Auth (AUTH_MODE=internal):
- `POST /api/auth/login` - Login with username/password
- `POST /api/auth/logout` - Logout
- `GET /api/auth/me` - Get current user
- `GET /api/users` - List users (admin only)
- `POST /api/users` - Create user (admin only)

Keycloak Auth (AUTH_MODE=keycloak):
- `GET /api/auth/keycloak/login` - Initiate OAuth2 flow
- `GET /api/auth/keycloak/callback` - OAuth2 callback handler
- `POST /api/auth/keycloak/logout` - Keycloak logout
- `GET /api/auth/info` - Get auth mode and user info

**Dependencies Added**:
- `github.com/coreos/go-oidc/v3/oidc` - OIDC client library
- `golang.org/x/oauth2` - OAuth2 client library

**Migration Path**:
- Existing deployments continue to work with `AUTH_MODE=internal`
- BiblioHub swarm deployments use `AUTH_MODE=keycloak`
- No data migration required - modes are independent
- E-reader OPDS access works in internal mode via Basic Auth
- E-reader OPDS access in OIDC mode also works via Basic Auth (see fix below)

**Impact**:
- Enables seamless integration with BiblioHub's centralized authentication
- Maintains backward compatibility for standalone deployments
- Provides flexible deployment options for different use cases
- Supports both web UI and e-reader clients

---

### fix/opds-basic-auth-in-oidc-mode - OPDS Basic Auth via Keycloak ROPC (2026-01-28)

**Problem**: When running in OIDC mode (`AUTH_MODE=oidc`), the OPDS feed endpoints rejected HTTP Basic Auth credentials, returning 401 Unauthorized. This broke:
- E-reader OPDS clients that use Basic Auth
- Service-to-service calls (e.g., abb_tts connecting to opds-server)

**Root Cause**: The `CheckSessionOrBasicAuth` method in `internal/auth/manager.go` only checked for OIDC session cookies in OIDC mode, ignoring Basic Auth headers entirely.

**Solution**: Implemented Keycloak ROPC (Resource Owner Password Credentials) authentication for Basic Auth in OIDC mode. This allows using the same Keycloak user accounts for both Web UI (OAuth2 flow) and OPDS feeds (Basic Auth).

**Keycloak Configuration** (in `biblio-hub/keycloak/biblio-realm.json`):
- Added `opds_user` realm role - required for OPDS feed access
- Added `opds_users` group with `opds_user` role
- Test users (`testadmin`, `testuser`) assigned to `opds_users` group
- `opds-server` client has `directAccessGrantsEnabled: true` (ROPC support)

**Files Changed**:
- `internal/auth/oidc.go`:
  - Added `AuthenticateWithPassword()` method using ROPC grant
  - Validates user has `opds_user` role for OPDS access
- `internal/auth/manager.go`:
  - Modified `CheckSessionOrBasicAuth()` to use OIDC ROPC for Basic Auth in OIDC mode

**Authentication Flow**:

| Mode | Web UI | OPDS Feeds |
|------|--------|------------|
| `internal` | Internal DB (session) | Internal DB (Basic Auth) |
| `oidc` | Keycloak (OAuth2 flow) | Keycloak (Basic Auth via ROPC) |

**OPDS Access Requirements** (OIDC mode):
1. User must exist in Keycloak
2. User must have `opds_user` role (via `opds_users` group or direct assignment)
3. User authenticates via HTTP Basic Auth with Keycloak credentials

**Benefits**:
- Single user database (Keycloak) for both Web UI and OPDS
- Centralized role management
- No need for separate opds-server internal users in OIDC mode

---

*Last updated: 2026-01-28*
