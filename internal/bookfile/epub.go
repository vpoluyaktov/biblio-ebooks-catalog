package bookfile

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

// EPUBMetadata represents metadata extracted from an EPUB file
type EPUBMetadata struct {
	Title       string
	Authors     []Author
	Language    string
	Description string
	Series      string
	SeriesIndex int
	Genres      []string
	CoverData   []byte
	CoverType   string
}

// Author represents a book author
type Author struct {
	FirstName  string
	LastName   string
	MiddleName string
}

// epubContainer represents the META-INF/container.xml structure
type epubContainer struct {
	XMLName  xml.Name `xml:"container"`
	RootFile struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

// epubPackage represents the content.opf structure
type epubPackage struct {
	XMLName  xml.Name     `xml:"package"`
	Metadata epubMetadata `xml:"metadata"`
	Manifest struct {
		Items []epubManifestItem `xml:"item"`
	} `xml:"manifest"`
}

// epubMetadata represents Dublin Core metadata with Calibre extensions
type epubMetadata struct {
	Titles      []string      `xml:"title"`
	Creators    []epubCreator `xml:"creator"`
	Languages   []string      `xml:"language"`
	Subjects    []string      `xml:"subject"`
	Description string        `xml:"description"`
	Metas       []epubMeta    `xml:"meta"`
}

// epubCreator represents a creator with optional file-as attribute
type epubCreator struct {
	Name   string `xml:",chardata"`
	FileAs string `xml:"file-as,attr"`
	Role   string `xml:"role,attr"`
}

// epubMeta represents calibre metadata extensions
type epubMeta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

// epubManifestItem represents a file in the EPUB
type epubManifestItem struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

// ParseEPUBMetadata extracts metadata from an EPUB file
func ParseEPUBMetadata(filePath string) (*EPUBMetadata, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open EPUB: %w", err)
	}
	defer r.Close()

	return parseEPUBFromZip(&r.Reader)
}

// ParseEPUBMetadataFromReader extracts metadata from an EPUB reader
func ParseEPUBMetadataFromReader(r io.ReaderAt, size int64) (*EPUBMetadata, error) {
	zipReader, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("failed to open EPUB as zip: %w", err)
	}

	return parseEPUBFromZip(zipReader)
}

func parseEPUBFromZip(zr *zip.Reader) (*EPUBMetadata, error) {
	// Find and parse container.xml
	containerFile, err := findFileInZip(zr, "META-INF/container.xml")
	if err != nil {
		return nil, fmt.Errorf("container.xml not found: %w", err)
	}

	var container epubContainer
	if err := parseXMLFromZipFile(containerFile, &container); err != nil {
		return nil, fmt.Errorf("failed to parse container.xml: %w", err)
	}

	// Find and parse the package file (content.opf)
	packageFile, err := findFileInZip(zr, container.RootFile.FullPath)
	if err != nil {
		return nil, fmt.Errorf("package file not found: %w", err)
	}

	var pkg epubPackage
	if err := parseXMLFromZipFile(packageFile, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package file: %w", err)
	}

	// Extract metadata
	metadata := &EPUBMetadata{}

	// Title
	if len(pkg.Metadata.Titles) > 0 {
		metadata.Title = strings.TrimSpace(pkg.Metadata.Titles[0])
	}

	// Authors
	metadata.Authors = parseEPUBAuthors(pkg.Metadata.Creators)

	// Language
	if len(pkg.Metadata.Languages) > 0 {
		lang := strings.TrimSpace(pkg.Metadata.Languages[0])
		if len(lang) > 2 {
			lang = lang[:2]
		}
		metadata.Language = lang
	}

	// Description
	metadata.Description = strings.TrimSpace(pkg.Metadata.Description)
	if metadata.Description == "" && len(pkg.Metadata.Subjects) > 0 {
		metadata.Description = strings.Join(pkg.Metadata.Subjects, ", ")
	}

	// Series and genres from Calibre metadata
	for _, meta := range pkg.Metadata.Metas {
		switch meta.Name {
		case "calibre:series":
			metadata.Series = strings.TrimSpace(meta.Content)
		case "calibre:series_index":
			fmt.Sscanf(meta.Content, "%d", &metadata.SeriesIndex)
		}
	}

	// Genres from subjects
	metadata.Genres = pkg.Metadata.Subjects

	// Extract cover image
	baseDir := filepath.Dir(container.RootFile.FullPath)
	coverHref := extractCoverHref(pkg, baseDir)
	if coverHref != "" {
		coverFile, err := findFileInZip(zr, coverHref)
		if err == nil {
			rc, err := coverFile.Open()
			if err == nil {
				defer rc.Close()
				coverData, err := io.ReadAll(rc)
				if err == nil {
					metadata.CoverData = coverData
					if strings.HasSuffix(strings.ToLower(coverHref), ".png") {
						metadata.CoverType = "image/png"
					} else {
						metadata.CoverType = "image/jpeg"
					}
				}
			}
		}
	}

	return metadata, nil
}

// parseEPUBAuthors extracts author information from EPUB creators
func parseEPUBAuthors(creators []epubCreator) []Author {
	var authors []Author

	for _, creator := range creators {
		// Skip if not an author (role might be editor, illustrator, etc.)
		if creator.Role != "" && creator.Role != "aut" {
			continue
		}

		name := strings.TrimSpace(creator.Name)
		if name == "" {
			continue
		}

		author := Author{}

		// Try to parse "LastName, FirstName" format
		if strings.Contains(name, ",") {
			parts := strings.SplitN(name, ",", 2)
			author.LastName = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				// FirstName might contain middle name
				nameParts := strings.Fields(strings.TrimSpace(parts[1]))
				if len(nameParts) > 0 {
					author.FirstName = nameParts[0]
				}
				if len(nameParts) > 1 {
					author.MiddleName = strings.Join(nameParts[1:], " ")
				}
			}
		} else {
			// Try to parse "FirstName LastName" format
			nameParts := strings.Fields(name)
			if len(nameParts) == 1 {
				author.LastName = nameParts[0]
			} else if len(nameParts) == 2 {
				author.FirstName = nameParts[0]
				author.LastName = nameParts[1]
			} else if len(nameParts) > 2 {
				author.FirstName = nameParts[0]
				author.MiddleName = strings.Join(nameParts[1:len(nameParts)-1], " ")
				author.LastName = nameParts[len(nameParts)-1]
			}
		}

		if author.LastName != "" || author.FirstName != "" {
			authors = append(authors, author)
		}
	}

	return authors
}

// extractCoverHref finds the cover image href from the EPUB package
func extractCoverHref(pkg epubPackage, baseDir string) string {
	// Look for items that might be cover images
	for _, item := range pkg.Manifest.Items {
		id := strings.ToLower(item.ID)
		href := strings.ToLower(item.Href)
		if (strings.Contains(id, "cover") || strings.Contains(href, "cover")) &&
			(item.MediaType == "image/jpeg" || item.MediaType == "image/png" ||
				item.MediaType == "image/jpg") {
			return filepath.Join(baseDir, item.Href)
		}
	}

	return ""
}

// findFileInZip finds a file in the ZIP archive
func findFileInZip(zr *zip.Reader, name string) (*zip.File, error) {
	for _, f := range zr.File {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

// parseXMLFromZipFile parses XML from a ZIP file
func parseXMLFromZipFile(f *zip.File, v interface{}) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	return xml.Unmarshal(data, v)
}

// ExtractEPUBCover extracts the cover image from an EPUB file
func ExtractEPUBCover(filePath string) ([]byte, string, error) {
	metadata, err := ParseEPUBMetadata(filePath)
	if err != nil {
		return nil, "", err
	}

	if metadata.CoverData == nil {
		return nil, "", fmt.Errorf("no cover found")
	}

	return metadata.CoverData, metadata.CoverType, nil
}

// fb2MetadataDocument extends the existing fb2Document with additional metadata fields
type fb2MetadataDocument struct {
	XMLName     xml.Name `xml:"FictionBook"`
	Description struct {
		TitleInfo struct {
			fb2TitleInfo
			Author struct {
				FirstName  string `xml:"first-name"`
				LastName   string `xml:"last-name"`
				MiddleName string `xml:"middle-name"`
			} `xml:"author"`
			BookTitle string   `xml:"book-title"`
			Genres    []string `xml:"genre"`
			Lang      string   `xml:"lang"`
			Sequence  struct {
				Name   string `xml:"name,attr"`
				Number int    `xml:"number,attr"`
			} `xml:"sequence"`
		} `xml:"title-info"`
	} `xml:"description"`
	Binaries []fb2Binary `xml:"binary"`
}

// ParseFB2Metadata extracts metadata from a FB2 file
func ParseFB2Metadata(filePath string) (*EPUBMetadata, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return parseFB2Metadata(f)
}

// ParseFB2MetadataFromZip extracts metadata from a FB2 file inside a ZIP archive
func ParseFB2MetadataFromZip(zipPath string) (*EPUBMetadata, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP: %w", err)
	}
	defer r.Close()

	// Find the FB2 file (should be the first .fb2 file)
	var fb2File *zip.File
	for _, f := range r.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".fb2") {
			fb2File = f
			break
		}
	}

	if fb2File == nil {
		return nil, fmt.Errorf("no FB2 file found in archive")
	}

	rc, err := fb2File.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open FB2 file: %w", err)
	}
	defer rc.Close()

	return parseFB2Metadata(rc)
}

// charsetReader returns a reader for the specified charset
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	charset = strings.ToLower(charset)

	switch charset {
	case "windows-1251":
		return charmap.Windows1251.NewDecoder().Reader(input), nil
	case "windows-1252":
		return charmap.Windows1252.NewDecoder().Reader(input), nil
	case "iso-8859-1", "latin1":
		return charmap.ISO8859_1.NewDecoder().Reader(input), nil
	case "utf-8", "":
		return input, nil
	case "utf-16", "utf-16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder().Reader(input), nil
	case "utf-16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder().Reader(input), nil
	default:
		// For unknown charsets, try to parse as UTF-8
		return input, nil
	}
}

func parseFB2Metadata(r io.Reader) (*EPUBMetadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read FB2: %w", err)
	}

	var fb2 fb2MetadataDocument
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charsetReader

	if err := decoder.Decode(&fb2); err != nil {
		return nil, fmt.Errorf("failed to parse FB2: %w", err)
	}

	// Get annotation from paragraphs
	annotation := strings.Join(fb2.Description.TitleInfo.Annotation.Paragraphs, "\n\n")

	metadata := &EPUBMetadata{
		Title:       strings.TrimSpace(fb2.Description.TitleInfo.BookTitle),
		Language:    strings.TrimSpace(fb2.Description.TitleInfo.Lang),
		Description: strings.TrimSpace(annotation),
		Series:      strings.TrimSpace(fb2.Description.TitleInfo.Sequence.Name),
		SeriesIndex: fb2.Description.TitleInfo.Sequence.Number,
		Genres:      fb2.Description.TitleInfo.Genres,
	}

	// Parse author
	author := Author{
		FirstName:  strings.TrimSpace(fb2.Description.TitleInfo.Author.FirstName),
		LastName:   strings.TrimSpace(fb2.Description.TitleInfo.Author.LastName),
		MiddleName: strings.TrimSpace(fb2.Description.TitleInfo.Author.MiddleName),
	}
	if author.FirstName != "" || author.LastName != "" {
		metadata.Authors = []Author{author}
	}

	// Extract cover image
	var coverID string
	for _, img := range fb2.Description.TitleInfo.Coverpage.Images {
		href := img.Href
		if href == "" {
			href = img.XlinkHref
		}
		if href == "" {
			href = img.LHref
		}
		if href != "" {
			coverID = strings.TrimPrefix(href, "#")
			break
		}
	}

	if coverID != "" {
		for _, binary := range fb2.Binaries {
			if binary.ID == coverID {
				decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(binary.Data))
				if err == nil {
					metadata.CoverData = decoded
					metadata.CoverType = binary.ContentType
					if metadata.CoverType == "" {
						if bytes.HasPrefix(decoded, []byte{0xFF, 0xD8, 0xFF}) {
							metadata.CoverType = "image/jpeg"
						} else if bytes.HasPrefix(decoded, []byte{0x89, 0x50, 0x4E, 0x47}) {
							metadata.CoverType = "image/png"
						} else {
							metadata.CoverType = "image/jpeg"
						}
					}
				}
				break
			}
		}
	}

	return metadata, nil
}

// ParseFB2MetadataFromBytes parses FB2 metadata from a byte array
func ParseFB2MetadataFromBytes(data []byte) (*EPUBMetadata, error) {
	var fb2 fb2MetadataDocument
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charsetReader

	if err := decoder.Decode(&fb2); err != nil {
		return nil, fmt.Errorf("failed to parse FB2: %w", err)
	}

	// Get annotation from paragraphs
	annotation := strings.Join(fb2.Description.TitleInfo.Annotation.Paragraphs, "\n\n")

	metadata := &EPUBMetadata{
		Title:       strings.TrimSpace(fb2.Description.TitleInfo.BookTitle),
		Language:    strings.TrimSpace(fb2.Description.TitleInfo.Lang),
		Description: strings.TrimSpace(annotation),
		Series:      strings.TrimSpace(fb2.Description.TitleInfo.Sequence.Name),
		SeriesIndex: fb2.Description.TitleInfo.Sequence.Number,
		Genres:      fb2.Description.TitleInfo.Genres,
	}

	// Parse author
	author := Author{
		FirstName:  strings.TrimSpace(fb2.Description.TitleInfo.Author.FirstName),
		LastName:   strings.TrimSpace(fb2.Description.TitleInfo.Author.LastName),
		MiddleName: strings.TrimSpace(fb2.Description.TitleInfo.Author.MiddleName),
	}
	if author.FirstName != "" || author.LastName != "" {
		metadata.Authors = []Author{author}
	}

	return metadata, nil
}
