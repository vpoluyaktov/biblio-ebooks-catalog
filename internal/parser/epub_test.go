package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEPUBMetadata(t *testing.T) {
	// Check if test data exists
	testDir := "../../testdata/epub_library"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("Test EPUB library not found, skipping test")
	}

	// Find first EPUB file in test directory
	var testFile string
	err := filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".epub" {
			testFile = path
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk test directory: %v", err)
	}

	if testFile == "" {
		t.Skip("No EPUB files found in test directory")
	}

	// Parse metadata
	metadata, err := ParseEPUBMetadata(testFile)
	if err != nil {
		t.Fatalf("Failed to parse EPUB: %v", err)
	}

	// Verify basic metadata exists
	if metadata.Title == "" {
		t.Error("Expected non-empty title")
	}

	if len(metadata.Authors) == 0 {
		t.Error("Expected at least one author")
	}

	t.Logf("Parsed EPUB: %s by %v", metadata.Title, metadata.Authors)
	t.Logf("Language: %s", metadata.Language)
	t.Logf("Series: %s (index %d)", metadata.Series, metadata.SeriesIndex)
	t.Logf("Genres: %v", metadata.Genres)
	if metadata.CoverData != nil {
		t.Logf("Cover: %d bytes (%s)", len(metadata.CoverData), metadata.CoverType)
	}
}

func TestParseEPUBAuthors(t *testing.T) {
	tests := []struct {
		name     string
		creators []epubCreator
		expected []Author
	}{
		{
			name: "LastName, FirstName format",
			creators: []epubCreator{
				{Name: "Doe, John", Role: "aut"},
			},
			expected: []Author{
				{FirstName: "John", LastName: "Doe"},
			},
		},
		{
			name: "FirstName LastName format",
			creators: []epubCreator{
				{Name: "John Doe", Role: "aut"},
			},
			expected: []Author{
				{FirstName: "John", LastName: "Doe"},
			},
		},
		{
			name: "FirstName MiddleName LastName format",
			creators: []epubCreator{
				{Name: "John Michael Doe", Role: "aut"},
			},
			expected: []Author{
				{FirstName: "John", MiddleName: "Michael", LastName: "Doe"},
			},
		},
		{
			name: "Skip non-authors",
			creators: []epubCreator{
				{Name: "John Doe", Role: "edt"},
				{Name: "Jane Smith", Role: "aut"},
			},
			expected: []Author{
				{FirstName: "Jane", LastName: "Smith"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEPUBAuthors(tt.creators)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d authors, got %d", len(tt.expected), len(result))
				return
			}
			for i, author := range result {
				if author.FirstName != tt.expected[i].FirstName ||
					author.LastName != tt.expected[i].LastName ||
					author.MiddleName != tt.expected[i].MiddleName {
					t.Errorf("Author %d: expected %+v, got %+v", i, tt.expected[i], author)
				}
			}
		})
	}
}
