package parser

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

type epubTOCEntry struct {
	Title  string
	Path   string
	Anchor string
}

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

func extractEPUBContent(reader io.ReaderAt, size int64) (*BookContent, error) {
	zr, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to open EPUB: %w", err)
	}

	containerFile, err := findFileInZip(zr, "META-INF/container.xml")
	if err != nil {
		return nil, fmt.Errorf("container.xml not found: %w", err)
	}

	var container epubContainer
	if err := parseXMLFromZipFile(containerFile, &container); err != nil {
		return nil, fmt.Errorf("failed to parse container.xml: %w", err)
	}

	packageFile, err := findFileInZip(zr, container.RootFile.FullPath)
	if err != nil {
		return nil, fmt.Errorf("package file not found: %w", err)
	}

	var pkg struct {
		XMLName  xml.Name `xml:"package"`
		Metadata struct {
			Titles   []string `xml:"title"`
			Creators []struct {
				Name string `xml:",chardata"`
			} `xml:"creator"`
		} `xml:"metadata"`
		Manifest struct {
			Items []struct {
				ID        string `xml:"id,attr"`
				Href      string `xml:"href,attr"`
				MediaType string `xml:"media-type,attr"`
			} `xml:"item"`
		} `xml:"manifest"`
		Spine struct {
			TOC      string `xml:"toc,attr"`
			ItemRefs []struct {
				IDRef string `xml:"idref,attr"`
			} `xml:"itemref"`
		} `xml:"spine"`
	}

	if err := parseXMLFromZipFile(packageFile, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package file: %w", err)
	}

	content := &BookContent{Format: "epub", Chapters: []Chapter{}}
	if len(pkg.Metadata.Titles) > 0 {
		content.Title = strings.TrimSpace(pkg.Metadata.Titles[0])
	}
	if len(pkg.Metadata.Creators) > 0 {
		content.Author = strings.TrimSpace(pkg.Metadata.Creators[0].Name)
	}

	manifestMap := make(map[string]string)
	manifestMediaTypeMap := make(map[string]string)
	for _, item := range pkg.Manifest.Items {
		manifestMap[item.ID] = item.Href
		manifestMediaTypeMap[item.ID] = item.MediaType
	}

	baseDir := filepath.Dir(container.RootFile.FullPath)
	if tocChapters := extractChaptersFromEPUBTOC(zr, baseDir, manifestMap, manifestMediaTypeMap, pkg.Spine.TOC); len(tocChapters) > 0 {
		content.Chapters = tocChapters
		return content, nil
	}

	for i, itemRef := range pkg.Spine.ItemRefs {
		href, ok := manifestMap[itemRef.IDRef]
		if !ok {
			continue
		}

		fullPath := normalizeEPUBPath(baseDir, href)
		chapterFile, err := findFileInZip(zr, fullPath)
		if err != nil {
			continue
		}

		rc, err := chapterFile.Open()
		if err != nil {
			continue
		}

		chapterData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		htmlContent := string(chapterData)
		defaultTitle := fmt.Sprintf("Chapter %d", i+1)
		chapterTitle := extractEPUBChapterTitle(htmlContent, defaultTitle)

		content.Chapters = append(content.Chapters, Chapter{
			ID:      itemRef.IDRef,
			Title:   strings.TrimSpace(chapterTitle),
			Content: htmlContent,
		})
	}

	return content, nil
}

func extractChaptersFromEPUBTOC(zr *zip.Reader, packageBaseDir string, manifestMap map[string]string, manifestMediaTypeMap map[string]string, spineTOCID string) []Chapter {
	entries := extractEPUBTOCEntries(zr, packageBaseDir, manifestMap, manifestMediaTypeMap, spineTOCID)
	if len(entries) == 0 {
		return nil
	}

	htmlCache := make(map[string]string)
	chapters := make([]Chapter, 0, len(entries))
	for i, entry := range entries {
		if entry.Path == "" || strings.TrimSpace(entry.Title) == "" {
			continue
		}

		htmlContent, ok := htmlCache[entry.Path]
		if !ok {
			chapterFile, err := findFileInZip(zr, entry.Path)
			if err != nil {
				continue
			}
			rc, err := chapterFile.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			htmlContent = string(data)
			htmlCache[entry.Path] = htmlContent
		}

		start := findEPUBAnchorStart(htmlContent, entry.Anchor)
		end := len(htmlContent)
		if i+1 < len(entries) && entries[i+1].Path == entry.Path {
			nextStart := findEPUBAnchorStart(htmlContent, entries[i+1].Anchor)
			if nextStart > start {
				end = nextStart
			}
		}
		if start < 0 || start >= len(htmlContent) {
			start = 0
		}
		if end <= start || end > len(htmlContent) {
			end = len(htmlContent)
		}

		segment := strings.TrimSpace(htmlContent[start:end])
		if segment == "" {
			continue
		}

		title := strings.TrimSpace(entry.Title)
		title = extractEPUBChapterTitle(segment, title)
		chapters = append(chapters, Chapter{
			ID:      fmt.Sprintf("toc-%d", i+1),
			Title:   title,
			Content: segment,
		})
	}

	return chapters
}

func extractEPUBTOCEntries(zr *zip.Reader, packageBaseDir string, manifestMap map[string]string, manifestMediaTypeMap map[string]string, spineTOCID string) []epubTOCEntry {
	tocIDs := make([]string, 0, 4)
	if spineTOCID != "" {
		tocIDs = append(tocIDs, spineTOCID)
	}
	for id, mediaType := range manifestMediaTypeMap {
		if mediaType == "application/x-dtbncx+xml" || (mediaType == "application/xhtml+xml" && strings.Contains(strings.ToLower(id), "nav")) {
			tocIDs = append(tocIDs, id)
		}
	}

	for _, tocID := range tocIDs {
		tocHref, ok := manifestMap[tocID]
		if !ok {
			continue
		}
		tocPath := normalizeEPUBPath(packageBaseDir, tocHref)
		tocFile, err := findFileInZip(zr, tocPath)
		if err != nil {
			continue
		}

		mediaType := manifestMediaTypeMap[tocID]
		tocBaseDir := filepath.Dir(tocPath)
		if mediaType == "application/x-dtbncx+xml" {
			entries, err := parseNCXTOCEntriesFromZipFile(tocFile, tocBaseDir)
			if err == nil && len(entries) > 0 {
				return entries
			}
			continue
		}
		if mediaType == "application/xhtml+xml" {
			entries, err := parseNavXHTMLTOCEntriesFromZipFile(tocFile, tocBaseDir)
			if err == nil && len(entries) > 0 {
				return entries
			}
		}
	}

	return nil
}

func parseNCXTOCEntriesFromZipFile(f *zip.File, tocBaseDir string) ([]epubTOCEntry, error) {
	var ncx struct {
		NavMap struct {
			NavPoints []ncxNavPoint `xml:"navPoint"`
		} `xml:"navMap"`
	}
	if err := parseXMLFromZipFile(f, &ncx); err != nil {
		return nil, err
	}

	entries := make([]epubTOCEntry, 0, len(ncx.NavMap.NavPoints))
	collectNCXTOCEntries(ncx.NavMap.NavPoints, tocBaseDir, &entries)
	return entries, nil
}

type ncxNavPoint struct {
	NavLabel struct {
		Text string `xml:"text"`
	} `xml:"navLabel"`
	Content struct {
		Src string `xml:"src,attr"`
	} `xml:"content"`
	NavPoints []ncxNavPoint `xml:"navPoint"`
}

func collectNCXTOCEntries(points []ncxNavPoint, tocBaseDir string, out *[]epubTOCEntry) {
	for _, point := range points {
		title := strings.TrimSpace(stripHTMLTagsForContent(point.NavLabel.Text))
		src := strings.TrimSpace(point.Content.Src)
		if title != "" && src != "" {
			filePath, anchor := splitEPUBHref(src)
			*out = append(*out, epubTOCEntry{
				Title:  title,
				Path:   normalizeEPUBPath(tocBaseDir, filePath),
				Anchor: anchor,
			})
		}
		if len(point.NavPoints) > 0 {
			collectNCXTOCEntries(point.NavPoints, tocBaseDir, out)
		}
	}
}

func parseNavXHTMLTOCEntriesFromZipFile(f *zip.File, tocBaseDir string) ([]epubTOCEntry, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	// Lenient fallback parser for nav.xhtml when XML namespaces are inconsistent.
	re := regexp.MustCompile(`(?is)<a[^>]*href\s*=\s*"([^"]+)"[^>]*>(.*?)</a>`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	entries := make([]epubTOCEntry, 0, len(matches))
	for _, m := range matches {
		href := strings.TrimSpace(m[1])
		title := strings.TrimSpace(stripHTMLTagsForContent(m[2]))
		if href == "" || title == "" {
			continue
		}
		filePath, anchor := splitEPUBHref(href)
		entries = append(entries, epubTOCEntry{
			Title:  title,
			Path:   normalizeEPUBPath(tocBaseDir, filePath),
			Anchor: anchor,
		})
	}

	return entries, nil
}

func extractEPUBChapterTitle(htmlContent, fallback string) string {
	headingPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`),
		regexp.MustCompile(`(?is)<h2[^>]*>(.*?)</h2>`),
	}
	for _, pattern := range headingPatterns {
		matches := pattern.FindStringSubmatch(htmlContent)
		if len(matches) < 2 {
			continue
		}
		title := strings.TrimSpace(stripHTMLTagsForContent(matches[1]))
		if title != "" {
			return title
		}
	}

	titlePattern := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	titleMatches := titlePattern.FindStringSubmatch(htmlContent)
	if len(titleMatches) >= 2 {
		title := strings.TrimSpace(stripHTMLTagsForContent(titleMatches[1]))
		if title != "" {
			return title
		}
	}

	return fallback
}

func findEPUBAnchorStart(htmlContent, anchor string) int {
	if anchor == "" {
		return 0
	}
	quotedAnchor := regexp.QuoteMeta(anchor)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?is)<[^>]*\sid\s*=\s*"` + quotedAnchor + `"[^>]*>`),
		regexp.MustCompile(`(?is)<[^>]*\sname\s*=\s*"` + quotedAnchor + `"[^>]*>`),
		regexp.MustCompile(`(?is)<[^>]*\sid\s*=\s*'` + quotedAnchor + `'[^>]*>`),
		regexp.MustCompile(`(?is)<[^>]*\sname\s*=\s*'` + quotedAnchor + `'[^>]*>`),
	}
	for _, pattern := range patterns {
		loc := pattern.FindStringIndex(htmlContent)
		if loc != nil {
			return loc[0]
		}
	}
	return 0
}

func splitEPUBHref(href string) (string, string) {
	href = strings.TrimSpace(href)
	if href == "" {
		return "", ""
	}
	parts := strings.SplitN(href, "#", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.TrimSpace(parts[1])
}

func normalizeEPUBPath(baseDir, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if i := strings.Index(href, "?"); i >= 0 {
		href = href[:i]
	}
	return filepath.ToSlash(filepath.Clean(filepath.Join(baseDir, href)))
}

func stripHTMLTagsForContent(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
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

// EPUBParser implements the Parser interface for EPUB files
type EPUBParser struct{}

// Parse extracts metadata from an EPUB file
func (p *EPUBParser) Parse(filePath string) (*Metadata, error) {
	epubMeta, err := ParseEPUBMetadata(filePath)
	if err != nil {
		return nil, err
	}
	return epubMetadataToMetadata(epubMeta), nil
}

// ParseFromBytes extracts metadata from EPUB bytes
func (p *EPUBParser) ParseFromBytes(data []byte) (*Metadata, error) {
	return nil, fmt.Errorf("ParseFromBytes not supported for EPUB (requires ZIP structure)")
}

// ParseFromReader extracts metadata from an EPUB reader
func (p *EPUBParser) ParseFromReader(reader io.Reader) (*Metadata, error) {
	return nil, fmt.Errorf("ParseFromReader not supported for EPUB (requires ReaderAt)")
}

// Format returns the format this parser handles
func (p *EPUBParser) Format() string {
	return "epub"
}

// epubMetadataToMetadata converts EPUBMetadata to the generic Metadata type
func epubMetadataToMetadata(epub *EPUBMetadata) *Metadata {
	authors := make([]string, len(epub.Authors))
	for i, author := range epub.Authors {
		authors[i] = author.FullName()
	}

	return &Metadata{
		Title:       epub.Title,
		Authors:     authors,
		Language:    epub.Language,
		Description: epub.Description,
		Genres:      epub.Genres,
		Series:      epub.Series,
		SeriesIndex: epub.SeriesIndex,
		CoverData:   epub.CoverData,
		CoverType:   epub.CoverType,
	}
}

// FullName returns the full name of an author
func (a Author) FullName() string {
	parts := []string{}
	if a.FirstName != "" {
		parts = append(parts, a.FirstName)
	}
	if a.MiddleName != "" {
		parts = append(parts, a.MiddleName)
	}
	if a.LastName != "" {
		parts = append(parts, a.LastName)
	}
	return strings.Join(parts, " ")
}
