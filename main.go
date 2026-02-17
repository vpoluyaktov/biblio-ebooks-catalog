package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"biblio-ebooks-catalog/internal/auth"
	"biblio-ebooks-catalog/internal/config"
	"biblio-ebooks-catalog/internal/db"
	"biblio-ebooks-catalog/internal/importer"
	"biblio-ebooks-catalog/internal/server"
)

// Run with: go run .

var version = "0.1.0"

func main() {
	// Commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("biblio-catalog %s\n", version)
			return
		case "import":
			runImport()
			return
		case "scan":
			runScanImport()
			return
		case "reindex":
			runReindex()
			return
		case "delete-library":
			runDeleteLibrary()
			return
		case "create-user":
			runCreateUser()
			return
		case "serve":
			// Continue to server
			os.Args = append(os.Args[:1], os.Args[2:]...)
		}
	}

	runServer()
}

func runServer() {
	configPath := flag.String("config", "", "Path to config file")
	port := flag.Int("port", 0, "Server port (overrides config)")
	dbPath := flag.String("db", "", "Database path (overrides config)")
	restart := flag.Bool("restart", false, "Kill existing process on the port before starting")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *port > 0 {
		cfg.Server.Port = *port
	}
	if *dbPath != "" {
		cfg.Database.Path = *dbPath
	}

	if *restart {
		killProcessOnPort(cfg.Server.Port)
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := database.LoadGenres(); err != nil {
		log.Printf("Warning: Failed to load genres: %v", err)
	}

	srv, err := server.New(cfg, database)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting biblio-catalog %s on http://%s", version, addr)
	log.Printf("Authentication mode: %s", cfg.Auth.Mode)

	if err := srv.Run(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func runImport() {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	inpxPath := fs.String("inpx", "", "Path to INPX file")
	libName := fs.String("name", "", "Library name")
	libPath := fs.String("path", "", "Path to books directory")
	dbPath := fs.String("db", "./data/library.db", "Database path")
	fs.Parse(os.Args[2:])

	if *inpxPath == "" {
		log.Fatal("--inpx is required")
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := database.LoadGenres(); err != nil {
		log.Printf("Warning: Failed to load genres: %v", err)
	}

	// Derive library name from INPX filename if not provided
	if *libName == "" {
		*libName = strings.TrimSuffix(filepath.Base(*inpxPath), filepath.Ext(*inpxPath))
	}

	// Derive library path from INPX directory if not provided
	if *libPath == "" {
		*libPath = filepath.Dir(*inpxPath)
	}

	imp := importer.New(database)
	if _, err := imp.ImportINPX(*inpxPath, *libName, *libPath, false); err != nil {
		log.Fatalf("Import failed: %v", err)
	}

	log.Println("Import completed successfully")
}

func runScanImport() {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	libName := fs.String("name", "", "Library name")
	libPath := fs.String("path", "", "Path to books directory")
	dbPath := fs.String("db", "./data/library.db", "Database path")
	workers := fs.Int("workers", 4, "Number of parallel workers for parsing")
	recreate := fs.Bool("recreate", false, "Delete existing library and reimport")
	fs.Parse(os.Args[2:])

	if *libPath == "" {
		log.Fatal("--path is required")
	}
	if *libName == "" {
		log.Fatal("--name is required")
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := database.LoadGenres(); err != nil {
		log.Printf("Warning: Failed to load genres: %v", err)
	}

	// Check if library already exists
	if *recreate {
		libraries, err := database.GetLibraries()
		if err == nil {
			for _, lib := range libraries {
				if lib.Path == *libPath {
					log.Printf("Deleting existing library '%s' (ID: %d)", lib.Name, lib.ID)
					if err := database.DeleteLibrary(lib.ID); err != nil {
						log.Fatalf("Failed to delete existing library: %v", err)
					}
					break
				}
			}
		}
	}

	// Scan directory
	log.Printf("Scanning directory: %s", *libPath)
	scanner := importer.NewScanner(*libPath, *workers)
	scanner.SetProgressCallback(func(current, total int, message string) {
		log.Printf("[%d/%d] %s", current, total, message)
	})

	books, err := scanner.ScanDirectory()
	if err != nil {
		log.Fatalf("Scan failed: %v", err)
	}

	log.Printf("Found %d books, starting import...", len(books))

	// Import scanned books
	imp := importer.New(database)
	imp.SetProgressCallback(func(current, total int, message string) {
		log.Printf("[%d/%d] %s", current, total, message)
	})

	if _, err := imp.ImportScannedBooks(books, *libName, *libPath, false); err != nil {
		log.Fatalf("Import failed: %v", err)
	}

	log.Println("Scan import completed successfully")
}

func runReindex() {
	fs := flag.NewFlagSet("reindex", flag.ExitOnError)
	libraryID := fs.Int64("library-id", 0, "Library ID to export")
	libraryName := fs.String("library-name", "", "Library name to export")
	output := fs.String("output", "", "Output INPX file path")
	dbPath := fs.String("db", "./data/library.db", "Database path")
	fs.Parse(os.Args[2:])

	if *libraryID == 0 && *libraryName == "" {
		log.Fatal("Either --library-id or --library-name is required")
	}
	if *output == "" {
		log.Fatal("--output is required")
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	writer := importer.NewINPXWriter(database)
	writer.SetProgressCallback(func(current, total int, message string) {
		log.Printf("[%d/%d] %s", current, total, message)
	})

	if *libraryID > 0 {
		if err := writer.ExportLibraryToINPX(*libraryID, *output); err != nil {
			log.Fatalf("Reindex failed: %v", err)
		}
	} else {
		if err := writer.ExportLibraryByNameToINPX(*libraryName, *output); err != nil {
			log.Fatalf("Reindex failed: %v", err)
		}
	}

	log.Printf("Reindex completed successfully. INPX file created: %s", *output)
}

func runDeleteLibrary() {
	fs := flag.NewFlagSet("delete-library", flag.ExitOnError)
	libraryID := fs.Int64("id", 0, "Library ID to delete")
	configPath := fs.String("config", "", "Path to config file")
	dbPath := fs.String("db", "", "Database path (overrides config)")
	fs.Parse(os.Args[2:])

	if *libraryID <= 0 {
		log.Fatal("Library ID is required. Usage: biblio-catalog delete-library --id <library_id>")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *dbPath != "" {
		cfg.Database.Path = *dbPath
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	lib, err := database.GetLibrary(*libraryID)
	if err != nil {
		log.Fatalf("Library with ID %d not found", *libraryID)
	}

	if err := database.DeleteLibrary(*libraryID); err != nil {
		log.Fatalf("Failed to delete library: %v", err)
	}

	log.Printf("Library '%s' (ID: %d) deleted successfully", lib.Name, lib.ID)
}

func runCreateUser() {
	fs := flag.NewFlagSet("create-user", flag.ExitOnError)
	username := fs.String("username", "", "Username")
	password := fs.String("password", "", "Password")
	role := fs.String("role", "readonly", "User role (admin or readonly)")
	configPath := fs.String("config", "", "Path to config file")
	dbPath := fs.String("db", "", "Database path (overrides config)")
	fs.Parse(os.Args[2:])

	if *username == "" || *password == "" {
		log.Fatal("Username and password are required. Usage: biblio-catalog create-user --username <user> --password <pass> [--role admin|readonly]")
	}

	if *role != "admin" && *role != "readonly" {
		log.Fatal("Role must be 'admin' or 'readonly'")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *dbPath != "" {
		cfg.Database.Path = *dbPath
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	authService := auth.New(database)
	user, err := authService.CreateUser(*username, *password, *role)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	log.Printf("User '%s' (ID: %d, role: %s) created successfully", user.Username, user.ID, user.Role)
}

func killProcessOnPort(port int) {
	log.Printf("Attempting to kill process on port %d", port)
	var cmd *exec.Cmd
	portStr := fmt.Sprintf("%d", port)

	switch runtime.GOOS {
	case "linux", "darwin":
		// Use lsof to find process on port, then kill it
		out, err := exec.Command("lsof", "-t", "-i", ":"+portStr).Output()
		if err != nil {
			// No process found on port
			return
		}
		pids := strings.TrimSpace(string(out))
		if pids == "" {
			return
		}
		// Kill each PID found
		for _, pid := range strings.Split(pids, "\n") {
			pid = strings.TrimSpace(pid)
			if pid != "" {
				cmd = exec.Command("kill", pid)
				if err := cmd.Run(); err != nil {
					log.Printf("Warning: failed to kill process %s: %v", pid, err)
				} else {
					log.Printf("Killed process %s on port %d", pid, port)
				}
			}
		}
	case "windows":
		// Use netstat to find process, then taskkill
		out, err := exec.Command("netstat", "-ano").Output()
		if err != nil {
			return
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, ":"+portStr) && strings.Contains(line, "LISTENING") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					pid := fields[len(fields)-1]
					cmd = exec.Command("taskkill", "/PID", pid, "/F")
					if err := cmd.Run(); err != nil {
						log.Printf("Warning: failed to kill process %s: %v", pid, err)
					} else {
						log.Printf("Killed process %s on port %d", pid, port)
					}
				}
			}
		}
	}
}
