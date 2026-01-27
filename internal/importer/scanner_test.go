package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDirectory(t *testing.T) {
	// Check if test data exists
	testDir := "../../testdata/epub_library"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("Test EPUB library not found, skipping test")
	}

	scanner := NewScanner(testDir, 2)

	// Track progress
	var progressCalls int
	scanner.SetProgressCallback(func(current, total int, message string) {
		progressCalls++
		t.Logf("Progress: %d/%d - %s", current, total, message)
	})

	books, err := scanner.ScanDirectory()
	if err != nil {
		t.Fatalf("Failed to scan directory: %v", err)
	}

	if len(books) == 0 {
		t.Error("Expected to find at least one book")
	}

	t.Logf("Found %d books", len(books))

	// Verify book data
	for i, book := range books {
		if i < 5 { // Log first 5 books
			t.Logf("Book %d: %s", i+1, book.FilePath)
			if book.Metadata != nil {
				t.Logf("  Title: %s", book.Metadata.Title)
				t.Logf("  Authors: %v", book.Metadata.Authors)
				t.Logf("  Format: %s", book.Format)
				t.Logf("  Size: %d bytes", book.Size)
			} else if book.ParseError != nil {
				t.Logf("  Parse error: %v", book.ParseError)
			}
		}

		// Verify basic fields
		if book.FilePath == "" {
			t.Errorf("Book %d has empty FilePath", i)
		}
		if book.Format == "" {
			t.Errorf("Book %d has empty Format", i)
		}
		if book.Size == 0 {
			t.Errorf("Book %d has zero Size", i)
		}
	}

	if progressCalls == 0 {
		t.Error("Expected progress callback to be called")
	}
}

func TestFindBookFiles(t *testing.T) {
	// Check if test data exists
	testDir := "../../testdata/epub_library"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("Test EPUB library not found, skipping test")
	}

	scanner := NewScanner(testDir, 1)
	files, err := scanner.findBookFiles()
	if err != nil {
		t.Fatalf("Failed to find book files: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected to find at least one book file")
	}

	t.Logf("Found %d files", len(files))

	// Verify all files have supported extensions
	for _, file := range files {
		ext := filepath.Ext(file)
		hasValidExt := ext == ".epub" || ext == ".fb2" ||
			(len(filepath.Base(file)) >= 8 && filepath.Base(file)[len(filepath.Base(file))-8:] == ".fb2.zip")
		if !hasValidExt {
			t.Errorf("Unexpected file extension: %s", file)
		}
	}
}

func TestParseFile(t *testing.T) {
	// Check if test data exists
	testDir := "../../testdata/epub_library"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("Test EPUB library not found, skipping test")
	}

	scanner := NewScanner(testDir, 1)

	// Find first EPUB file
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

	books := scanner.parseFile(testFile)
	if len(books) == 0 {
		t.Fatal("Expected at least one book")
	}

	book := books[0]
	t.Logf("Parsed: %s", book.FilePath)
	if book.Metadata != nil {
		t.Logf("  Title: %s", book.Metadata.Title)
		t.Logf("  Authors: %v", book.Metadata.Authors)
	} else if book.ParseError != nil {
		t.Logf("  Parse error: %v", book.ParseError)
	}

	// Verify basic fields
	if book.FilePath != testFile {
		t.Errorf("Expected FilePath %s, got %s", testFile, book.FilePath)
	}
	if book.Format != "epub" {
		t.Errorf("Expected Format 'epub', got '%s'", book.Format)
	}
	if book.Size == 0 {
		t.Error("Expected non-zero Size")
	}
}
