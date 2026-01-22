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

	"biblio-opds-server/internal/auth"
	"biblio-opds-server/internal/config"
	"biblio-opds-server/internal/db"
	"biblio-opds-server/internal/importer"
	"biblio-opds-server/internal/server"
)

// Run with: go run .

var version = "0.1.0"

func main() {
	// Commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("opds-server %s\n", version)
			return
		case "import":
			runImport()
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

	srv := server.New(cfg, database)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting opds-server %s on http://%s", version, addr)

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

func runDeleteLibrary() {
	fs := flag.NewFlagSet("delete-library", flag.ExitOnError)
	libraryID := fs.Int64("id", 0, "Library ID to delete")
	configPath := fs.String("config", "", "Path to config file")
	dbPath := fs.String("db", "", "Database path (overrides config)")
	fs.Parse(os.Args[2:])

	if *libraryID <= 0 {
		log.Fatal("Library ID is required. Usage: opds-server delete-library --id <library_id>")
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
		log.Fatal("Username and password are required. Usage: opds-server create-user --username <user> --password <pass> [--role admin|readonly]")
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
