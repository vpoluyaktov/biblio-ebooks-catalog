# Biblio Catalog

> Part of the [BiblioHub](https://github.com/vpoluyaktov/biblio-hub) application suite

<img width="1728" height="842" alt="Screenshot 2026-03-22 at 23 00 50" src="https://github.com/user-attachments/assets/a1b1bd88-7d0d-467a-98c7-1c457dae37f7" />


A lightweight OPDS (Open Publication Distribution System) catalog server for e-book libraries. Import EPUB and FB2 book collections with or without INPX index files and serve them via OPDS protocol to e-readers and reading apps. Written in Go with a vanilla JavaScript frontend.

**Live demo: [https://demo.bibliohub.org/catalog/](https://demo.bibliohub.org/catalog/)**

## Features

### OPDS Catalog
- **OPDS 1.2 compliant** feed generation
- Browse books by **authors**, **series**, **genres**, and **recent additions**
- **Search** across titles, authors, and series
- **Book covers** extracted from EPUB and FB2 files
- **Multiple download formats** support

### Web Administration UI
- **Modern responsive design** with dark/light theme toggle
- **Library management** — import, enable/disable, edit, delete libraries
- **User management** — create users, assign roles (admin/user), change passwords
- **Real-time import progress** with book count display
- **Server-side file browser** for selecting INPX files and library paths

### Library Import
- **INPX Import** — Fast import from INPX index files
- **Scan Import** — Direct directory scanning for EPUB and FB2 files (no INPX needed)
- **Reindex** — Export library metadata to INPX format from database
- **Supported formats**: EPUB, FB2, FB2.ZIP
- **Parallel processing** with configurable worker pools

## Technology Stack

- **Language**: Go 1.24+
- **Database**: SQLite
- **Frontend**: Vanilla JavaScript SPA, dark/light theme
- **E-book Parsing**: [biblio-ebook-parser](https://github.com/vpoluyaktov/biblio-ebook-parser) library
- **OPDS**: Atom/XML feed generation (OPDS 1.2)
- **Auth**: Biblio Auth integration or local authentication
- **Deployment**: Docker, Docker Swarm (via BiblioHub)

## Quick Start (Docker)

The recommended way to run is as part of the [BiblioHub](https://github.com/vpoluyaktov/biblio-hub) Docker Swarm stack:

```bash
git clone https://github.com/vpoluyaktov/biblio-hub.git
cd biblio-hub
cp .env.example .env
./scripts/start_stack.sh
```

Access at: `http://localhost:9900/catalog/`

### From Source

```bash
git clone https://github.com/vpoluyaktov/biblio-ebooks-catalog.git
cd biblio-ebooks-catalog
go build -o biblio-catalog .
./biblio-catalog
```

Requires Go 1.24+. The server starts on `http://0.0.0.0:9988` by default.

## CLI Commands

```bash
./biblio-catalog import --inpx /path/to/library.inpx --name "My Library" --path /path/to/books
./biblio-catalog scan --name "My EPUB Library" --path /path/to/epub/books --workers 4
./biblio-catalog reindex --library-id 1 --output /path/to/output.inpx
./biblio-catalog delete-library --id 1
./biblio-catalog create-user --username admin --password secret --role admin
```

## OPDS Access

Each library has a unique OPDS endpoint:

```
http://your-server:9988/opds/{library_id}
```

Configure your e-reader with this URL and your credentials.

## API Endpoints

### Authentication
- `POST /api/auth/login` — Login
- `POST /api/auth/logout` — Logout
- `GET /api/auth/me` — Current user info

### Libraries
- `GET /api/libraries` — List all libraries
- `POST /api/libraries` — Import library from INPX
- `POST /api/libraries/scan` — Scan and import directory
- `POST /api/libraries/reindex` — Export library to INPX
- `PUT /api/libraries/{id}` — Update library
- `DELETE /api/libraries/{id}` — Delete library

### Users (Admin only)
- `GET /api/users` — List users
- `POST /api/users` — Create user
- `PUT /api/users/{id}/role` — Change user role
- `DELETE /api/users/{id}` — Delete user

### OPDS Feeds
- `GET /opds/{lib}` — Library root catalog
- `GET /opds/{lib}/authors` — Authors index
- `GET /opds/{lib}/author/{id}` — Books by author
- `GET /opds/{lib}/series` — Series index
- `GET /opds/{lib}/genres` — Genres list
- `GET /opds/{lib}/new` — Recent books
- `GET /opds/{lib}/search` — Search books

## Project Structure

```
biblio-ebooks-catalog/
├── main.go                 # Entry point
├── internal/
│   ├── db/                 # Database layer (SQLite)
│   ├── importer/           # INPX import logic
│   ├── opds/               # OPDS feed generation
│   └── server/             # HTTP server, API handlers, middleware
├── web/
│   ├── static/             # CSS, JS (SPA)
│   └── templates/          # HTML templates
└── data/
    └── genres.json         # Genre code mappings
```

## License

MIT
