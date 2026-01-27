package importer

// FileType represents the type of book file discovered
type FileType int

const (
	FileTypeFB2 FileType = iota
	FileTypeEPUB
	FileTypeZIP
	FileTypeInZip
)

// FileInfo represents a discovered book file before parsing
type FileInfo struct {
	// File identification
	Path     string   // Full path to file (or ZIP file for InZip)
	RelPath  string   // Relative path from library root
	FileName string   // File name
	Type     FileType // File type (FB2, EPUB, ZIP, InZip)
	Size     int64    // File size in bytes

	// For InZip files only
	ZipPath      string // Path to parent ZIP file
	FileInZip    string // File name inside ZIP
	OffsetInZip  int64  // Offset in ZIP for efficient extraction
	SizeInZip    int64  // Uncompressed size of file in ZIP
}

// IsInZip returns true if this file is inside a ZIP archive
func (f *FileInfo) IsInZip() bool {
	return f.Type == FileTypeInZip
}

// GetFormat returns the format string for this file
func (f *FileInfo) GetFormat() string {
	switch f.Type {
	case FileTypeFB2:
		return "fb2"
	case FileTypeEPUB:
		return "epub"
	case FileTypeInZip:
		// Determine format from file extension inside ZIP
		if len(f.FileInZip) > 4 && f.FileInZip[len(f.FileInZip)-4:] == ".fb2" {
			return "fb2"
		}
		if len(f.FileInZip) > 5 && f.FileInZip[len(f.FileInZip)-5:] == ".epub" {
			return "epub"
		}
		return "unknown"
	default:
		return "unknown"
	}
}
