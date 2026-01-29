# Biblio Catalog

> Part of the [BiblioHub](https://github.com/vpoluyaktov/BiblioHub) application suite

A lightweight OPDS (Open Publication Distribution System) catalog server for e-book libraries, written in Go. Import your EPUB and FB2 book collections with or without INPX index files and serve them via OPDS protocol to e-readers and reading apps.

## Features

### OPDS Catalog
- **OPDS 1.2 compliant** feed generation
- Browse books by **authors**, **series**, **genres**, and **recent additions**
- **Search** across titles, authors, and series
- **Book covers** extracted from FB2 files
- **Multiple download formats** support
- **Basic authentication** for OPDS access

### Web Administration UI
- **Modern responsive design** with dark/light theme toggle
- **Library management** - import, enable/disable, edit, delete libraries
- **User management** - create users, assign roles (admin/user), change passwords
- **Real-time import progress** with book count display
- **Server-side file browser** for selecting INPX files and library paths
- **Per-library OPDS URLs** with click-to-copy functionality

### Library Import
- **Three import modes:**
  - **INPX Import** - Fast import from INPX index files (lib.rus.ec format)
  - **Scan Import** - Direct directory scanning for EPUB and FB2 files (no INPX needed)
  - **Reindex** - Export library metadata to INPX format from database
- **Supported formats:** EPUB, FB2, FB2.ZIP
- **Metadata extraction** from EPUB (Dublin Core + Calibre) and FB2 files
- **Parallel processing** with configurable worker pools for fast scanning
- **Progress tracking** during import with live book count
- **Recreate mode** - Delete and reimport existing libraries
- **ID reuse** - deleted library IDs are recycled for stable OPDS URLs

## Installation

### Option 1: Docker (Recommended)

```bash
git clone https://github.com/vpoluyaktov/biblio-ebooks-catalog.git
cd biblio-ebooks-catalog/docker

# Create directories for data and books
mkdir -p data books

# Copy your INPX and ZIP files to the books directory
cp /path/to/your/library.inpx books/
cp /path/to/your/*.zip books/

# Start the server
docker-compose up -d
```

The server will be available at `http://localhost:9988`.

**Volume mounts:**
- `./docker/data` - Database and configuration (persisted)
- `./docker/books` - Your book library (INPX and ZIP files)

### Option 2: Build from source

**Prerequisites:** Go 1.21 or later

```bash
git clone https://github.com/vpoluyaktov/biblio-ebooks-catalog.git
cd biblio-ebooks-catalog
go build -o biblio-catalog .
```

## Usage

### Start the server
```bash
./biblio-catalog
```

The server starts on `http://0.0.0.0:9988` by default.

### Command line options
```bash
./biblio-catalog --help
./biblio-catalog --restart  # Kill existing process on port and restart
```

### CLI Commands

#### Import from INPX (Fast)
Import a library using an existing INPX index file:
```bash
./biblio-catalog import \
  --inpx /path/to/library.inpx \
  --name "My Library" \
  --path /path/to/books \
  --db ./data/library.db
```

#### Scan Import (No INPX needed)
Scan a directory and import EPUB/FB2 files directly:
```bash
./biblio-catalog scan \
  --name "My EPUB Library" \
  --path /path/to/epub/books \
  --workers 4 \
  --db ./data/library.db
```

**Options:**
- `--workers` - Number of parallel workers for parsing (default: 4)
- `--recreate` - Delete existing library with same path and reimport

**Example with recreate:**
```bash
./biblio-catalog scan \
  --name "My Library" \
  --path /path/to/books \
  --recreate
```

#### Reindex (Export to INPX)
Export library metadata from database to INPX format:
```bash
# By library ID
./biblio-catalog reindex \
  --library-id 1 \
  --output /path/to/output.inpx

# By library name
./biblio-catalog reindex \
  --library-name "My Library" \
  --output /path/to/output.inpx
```

#### Other Commands
```bash
# Delete a library
./biblio-catalog delete-library --id 1

# Create a user
./biblio-catalog create-user \
  --username admin \
  --password secret \
  --role admin

# Show version
./biblio-catalog version
```

### First-time setup
1. Open `http://localhost:9988` in your browser
2. Create an admin account on first launch
3. Import your first library by selecting an INPX file and library path

### OPDS Access
Each library has a unique OPDS endpoint:
```
http://your-server:9988/opds/{library_id}
```

Configure your e-reader with this URL and your credentials.

## Project Structure

```
biblio-ebooks-catalog/
├── main.go                 # Entry point
├── internal/
│   ├── db/                 # Database layer (SQLite)
│   │   ├── db.go          # Database initialization
│   │   ├── models.go      # Data models
│   │   └── queries.go     # Database queries
│   ├── importer/          # INPX import logic
│   │   └── inpx.go        # INPX parser and importer
│   ├── opds/              # OPDS feed generation
│   │   └── feed.go        # Atom/OPDS XML generation
│   └── server/            # HTTP server
│       ├── server.go      # Server setup and routing
│       ├── handlers_api.go    # REST API handlers
│       ├── handlers_opds.go   # OPDS feed handlers
│       └── middleware.go      # Auth middleware
├── web/
│   ├── static/
│   │   ├── css/style.css  # Styles with theme support
│   │   └── js/app.js      # SPA JavaScript
│   └── templates/
│       └── index.html     # Main HTML template
└── data/
    └── genres.json        # Genre code mappings
```

## API Endpoints

### Authentication
- `POST /api/auth/login` - Login
- `POST /api/auth/logout` - Logout
- `GET /api/auth/me` - Current user info

### Libraries
- `GET /api/libraries` - List all libraries
- `GET /api/libraries/{id}` - Get library details
- `GET /api/libraries/{id}/stats` - Get library statistics
- `POST /api/libraries` - Import library from INPX
- `GET /api/libraries/import` - Import library (SSE progress)
- `POST /api/libraries/scan` - Scan and import directory (EPUB/FB2)
- `POST /api/libraries/reindex` - Export library to INPX format
- `PUT /api/libraries/{id}` - Update library
- `DELETE /api/libraries/{id}` - Delete library

### Users (Admin only)
- `GET /api/users` - List users
- `POST /api/users` - Create user
- `PUT /api/users/{id}/role` - Change user role
- `PUT /api/users/{id}/password` - Change password
- `DELETE /api/users/{id}` - Delete user

### OPDS Feeds
- `GET /opds/{lib}` - Library root catalog
- `GET /opds/{lib}/authors` - Authors index
- `GET /opds/{lib}/author/{id}` - Books by author
- `GET /opds/{lib}/series` - Series index
- `GET /opds/{lib}/series/{id}` - Books in series
- `GET /opds/{lib}/genres` - Genres list
- `GET /opds/{lib}/genre/{id}` - Books by genre
- `GET /opds/{lib}/new` - Recent books
- `GET /opds/{lib}/search` - Search books

## Configuration

The server uses SQLite for data storage. The database file `opds.db` is created in the current directory on first run.

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
