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
| `AUTH_MODE` | Authentication mode (`internal` or `biblio-auth`) | `biblio-auth` |
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
    - AUTH_MODE=biblio-auth
    - BIBLIO_AUTH_URL=http://biblio-auth:80/auth
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
- **Universal Book Import** (2026-01-26)
  - EPUB and FB2 import with or without INPX index file
  - Three modes: INPX Import (fast), Directory Scan (slow), Reindex (export to INPX)
  - Streaming import with progress tracking and cancellation support
  - Robust FB2 parser with XML sanitization for malformed files
  - Unified parser interface for all book formats
- **In-Browser Ebook Reader** (2026-01-31)
  - Full-featured reader for EPUB and FB2 formats
  - Customizable: font size, font family, line height, themes (light/dark/sepia)
  - Chapter navigation, keyboard shortcuts, touch gestures
  - Responsive design for desktop and mobile
- **Reading History** (2026-02-01)
  - Track last 10 books with reading position (chapter + scroll)
  - Quick resume from Reader dropdown in toolbar
  - Mobile "Continue Reading" menu
- **Biblio Auth Integration** (2026-02-02)
  - Replaced OIDC/Keycloak with lightweight Biblio Auth
  - JWT token validation via Biblio Auth API
  - OPDS Basic Auth validated via Biblio Auth
  - **Admin Role Detection** (2026-02-02): Fixed config icon visibility for admin users in biblio-auth mode by adding role field to `/api/auth/info` response based on user's groups
  - **Reader Authentication** (2026-02-02): Fixed reader endpoint to support biblio-auth mode by checking auth_token cookie in CheckSession method

### Future Enhancements

- OPDS2 JSON feed
- Format conversion (FB2 → EPUB)
- Server-side reading progress sync

## Authentication

Biblio Catalog supports two authentication modes:

### Internal Mode (`AUTH_MODE=internal`)
- Standalone deployment with local SQLite user database
- Session-based authentication with cookies
- HTTP Basic Auth support for e-readers (OPDS clients)
- User management via web UI

### Biblio Auth Mode (`AUTH_MODE=biblio-auth`)
- Integration with Biblio Auth for centralized authentication
- JWT token validation via Biblio Auth API
- Web UI redirects to Biblio Auth login page
- HTTP Basic Auth for e-readers (validated via Biblio Auth API)
- User management via Biblio Auth Admin Console

For integration details, see [biblio-auth/INTEGRATION_GUIDE.md](https://github.com/vpoluyaktov/biblio-auth/blob/main/INTEGRATION_GUIDE.md)

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

*Last updated: 2026-02-02*
