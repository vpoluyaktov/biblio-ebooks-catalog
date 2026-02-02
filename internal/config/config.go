package config

import (
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Library    LibraryConfig    `yaml:"library"`
	Auth       AuthConfig       `yaml:"auth"`
	BiblioAuth BiblioAuthConfig `yaml:"biblio_auth"`
	OPDS       OPDSConfig       `yaml:"opds"`
}

type ServerConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	BasePath string `yaml:"base_path"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type LibraryConfig struct {
	BooksPerPage int `yaml:"books_per_page"`
}

type AuthConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Mode     string `yaml:"mode"` // internal, oidc, or biblio-auth
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type BiblioAuthConfig struct {
	URL string `yaml:"url"`
}

type OPDSConfig struct {
	ShowCovers      bool `yaml:"show_covers"`
	ShowAnnotations bool `yaml:"show_annotations"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 9988,
		},
		Database: DatabaseConfig{
			Path: "./db/opds.db",
		},
		Library: LibraryConfig{
			BooksPerPage: 50,
		},
		Auth: AuthConfig{
			Enabled: false,
			Mode:    "internal",
			User:    "admin",
		},
		BiblioAuth: BiblioAuthConfig{
			URL: "http://biblio-auth:80",
		},
		OPDS: OPDSConfig{
			ShowCovers:      true,
			ShowAnnotations: true,
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	cfg.loadFromEnv()

	if err := cfg.ensureDirectories(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) loadFromEnv() {
	if v := os.Getenv("OPDS_SERVER_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("OPDS_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv("OPDS_BASE_PATH"); v != "" {
		c.Server.BasePath = v
	}
	if v := os.Getenv("OPDS_DATABASE_PATH"); v != "" {
		c.Database.Path = v
	}
	if v := os.Getenv("OPDS_AUTH_ENABLED"); v == "true" {
		c.Auth.Enabled = true
	}
	if v := os.Getenv("OPDS_AUTH_USER"); v != "" {
		c.Auth.User = v
	}
	if v := os.Getenv("OPDS_AUTH_PASSWORD"); v != "" {
		c.Auth.Password = v
	}
	if v := os.Getenv("AUTH_MODE"); v != "" {
		c.Auth.Mode = v
	}

	// Biblio Auth configuration
	if v := os.Getenv("BIBLIO_AUTH_URL"); v != "" {
		c.BiblioAuth.URL = v
	}
}

func (c *Config) ensureDirectories() error {
	dir := filepath.Dir(c.Database.Path)
	return os.MkdirAll(dir, 0755)
}
