package bookfile

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type BookFile struct {
	LibraryPath string
	Archive     string
	File        string
	Format      string
}

func New(libraryPath, archive, file, format string) *BookFile {
	return &BookFile{
		LibraryPath: libraryPath,
		Archive:     archive,
		File:        file,
		Format:      format,
	}
}

func (bf *BookFile) GetReader() (io.ReadCloser, int64, error) {
	if bf.Archive == "" {
		// Direct file access (not in archive)
		filePath := filepath.Join(bf.LibraryPath, bf.File+"."+bf.Format)
		f, err := os.Open(filePath)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to open file: %w", err)
		}
		stat, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, 0, err
		}
		return f, stat.Size(), nil
	}

	// File is inside a ZIP archive
	archivePath := filepath.Join(bf.LibraryPath, bf.Archive)
	return bf.extractFromZip(archivePath)
}

func (bf *BookFile) extractFromZip(archivePath string) (io.ReadCloser, int64, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open archive %s: %w", archivePath, err)
	}

	// Look for the file in the archive
	fileName := bf.File + "." + bf.Format
	fileNameFb2 := bf.File + ".fb2"

	for _, f := range r.File {
		name := f.Name
		// Handle different encodings (CP866 is common in Russian archives)
		if strings.EqualFold(name, fileName) || strings.EqualFold(name, fileNameFb2) ||
			strings.HasSuffix(strings.ToLower(name), "/"+strings.ToLower(fileName)) ||
			strings.HasSuffix(strings.ToLower(name), "/"+strings.ToLower(fileNameFb2)) {

			rc, err := f.Open()
			if err != nil {
				r.Close()
				return nil, 0, fmt.Errorf("failed to open file in archive: %w", err)
			}

			return &zipFileReader{
				ReadCloser: rc,
				archive:    r,
			}, int64(f.UncompressedSize64), nil
		}
	}

	// Try matching by file ID (common pattern: just the number)
	for _, f := range r.File {
		baseName := strings.TrimSuffix(filepath.Base(f.Name), filepath.Ext(f.Name))
		if baseName == bf.File {
			rc, err := f.Open()
			if err != nil {
				r.Close()
				return nil, 0, fmt.Errorf("failed to open file in archive: %w", err)
			}

			return &zipFileReader{
				ReadCloser: rc,
				archive:    r,
			}, int64(f.UncompressedSize64), nil
		}
	}

	r.Close()
	return nil, 0, fmt.Errorf("file %s not found in archive %s", fileName, archivePath)
}

type zipFileReader struct {
	io.ReadCloser
	archive *zip.ReadCloser
}

func (zfr *zipFileReader) Close() error {
	err1 := zfr.ReadCloser.Close()
	err2 := zfr.archive.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func GetMimeType(format string) string {
	format = strings.ToLower(format)
	switch format {
	case "fb2":
		return "application/x-fictionbook+xml"
	case "fb2.zip":
		return "application/x-fictionbook+xml+zip"
	case "epub":
		return "application/epub+zip"
	case "epub.zip":
		return "application/epub+zip"
	case "mobi":
		return "application/x-mobipocket-ebook"
	case "azw3":
		return "application/x-mobi8-ebook"
	case "pdf":
		return "application/pdf"
	case "pdf.zip":
		return "application/pdf"
	case "djvu":
		return "image/vnd.djvu"
	case "djvu.zip":
		return "image/vnd.djvu"
	case "txt":
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

func GetFileExtension(format string) string {
	format = strings.ToLower(format)
	switch format {
	case "fb2", "fb2.zip":
		return ".fb2"
	case "epub", "epub.zip":
		return ".epub"
	case "mobi":
		return ".mobi"
	case "azw3":
		return ".azw3"
	case "pdf", "pdf.zip":
		return ".pdf"
	case "djvu", "djvu.zip":
		return ".djvu"
	case "txt":
		return ".txt"
	default:
		return "." + format
	}
}
