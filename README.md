# OPDS Server

A lightweight OPDS (Open Publication Distribution System) catalog server for e-book libraries, written in Go. Import your FB2 book collections from INPX files and serve them via OPDS protocol to e-readers and reading apps.

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
- Import from **INPX** index files (lib.rus.ec format)
- Supports custom field mappings via `structure.info`
- **Progress tracking** during import with live book count
- **ID reuse** - deleted library IDs are recycled for stable OPDS URLs

## Installation

### Prerequisites
- Go 1.21 or later

### Build from source
```bash
git clone https://github.com/vpoluyaktov/opds-server.git
cd opds-server
go build -o opds-server .
```

## Usage

### Start the server
```bash
./opds-server
```

The server starts on `http://0.0.0.0:9988` by default.

### Command line options
```bash
./opds-server --help
./opds-server --restart  # Kill existing process on port and restart
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
opds-server/
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
- `GET /api/libraries/import` - Import library (SSE progress)
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
