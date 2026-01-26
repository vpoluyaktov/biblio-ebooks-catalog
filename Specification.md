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

- **EPUB Library Import** (2026-01-26)
  - Feature branch: `feature/EPUB_Import`
  - Goal: Import EPUB files similar to FB2 import functionality
  - Test data: Python script to fetch 100 EPUB books from Project Gutenberg
  - Tasks:
    - [x] Create test data fetcher script (`scripts/fetch_gutenberg_epubs.py`)
    - [ ] EPUB metadata extraction (title, author, series, description)
    - [ ] EPUB cover extraction
    - [ ] EPUB library import command
    - [ ] Integration with existing book serving infrastructure

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
