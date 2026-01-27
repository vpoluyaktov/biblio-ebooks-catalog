package parser

import (
	"testing"
)

// Test series number parsing with various invalid formats
func TestParseSeriesNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		// Valid numeric values
		{"valid integer", "5", 5},
		{"valid with spaces", "  10  ", 10},
		{"zero", "0", 1},
		{"negative", "-5", 1},

		// Date-like formats (year + issue)
		{"year and month", "1996 02", 1996},
		{"year and issue", "2001 10", 2001},
		{"year with text", "2005 09", 2005},
		{"year special", "2001 спецвыпуск", 2001},

		// Underscore/dash formats
		{"underscore format", "09_2", 9},
		{"dash format", "01-03", 1},

		// Standard-like formats
		{"standard format 1", "9126-93", 9126},
		{"standard format 2", "51904-2002", 51904},
		{"decimal format", "2.001-93", 2},
		{"decimal format 2", "34.601-90", 34},

		// Corrupted/invalid data
		{"corrupted russian", "« name=»Рассказы", 1},
		{"corrupted series", "« name=»Записки следователя", 1},
		{"corrupted name", "« name=»Ийон Тихий", 1},
		{"empty string", "", 0},
		{"only text", "спецвыпуск", 1},
		{"only symbols", "«»", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSeriesNumber(tt.input)
			if result != tt.expected {
				t.Errorf("parseSeriesNumber(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// Test XML sanitization for invalid UTF-8
func TestFixInvalidUTF8(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		valid bool
	}{
		{"valid utf8", []byte("Hello мир"), true},
		{"invalid byte sequence", []byte{0xFF, 0xFE, 0x41}, false},
		{"windows-1251 cyrillic", []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}, false}, // "Привет" in Windows-1251
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixInvalidUTF8(tt.input)
			if len(result) == 0 {
				t.Errorf("fixInvalidUTF8 returned empty result")
			}
		})
	}
}

// Test removal of illegal XML characters
func TestRemoveIllegalXMLChars(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"control char U+0001", []byte("Hello\x01World"), "Hello World"},
		{"control char U+0011", []byte("Test\x11Data"), "Test Data"},
		{"control char U+001B", []byte("Foo\x1BBar"), "Foo Bar"},
		{"control char U+001E", []byte("A\x1EB"), "A B"},
		{"valid tab", []byte("A\tB"), "A\tB"},
		{"valid newline", []byte("A\nB"), "A\nB"},
		{"valid carriage return", []byte("A\rB"), "A\rB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeIllegalXMLChars(tt.input)
			resultStr := string(result)
			if resultStr != tt.expected {
				t.Errorf("removeIllegalXMLChars(%q) = %q, want %q", tt.input, resultStr, tt.expected)
			}
		})
	}
}

// Test unescaped ampersand fixing
func TestFixUnescapedAmpersands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid entity amp", "A &amp; B", "A &amp; B"},
		{"valid entity lt", "A &lt; B", "A &lt; B"},
		{"valid entity gt", "A &gt; B", "A &gt; B"},
		{"valid numeric entity", "A &#65; B", "A &#65; B"},
		{"valid hex entity", "A &#x41; B", "A &#x41; B"},
		{"unescaped ampersand", "A & B", "A &amp; B"},
		{"unescaped with text", "A &M B", "A &amp;M B"},
		{"unescaped no semicolon", "A &nbsp B", "A &amp;nbsp B"},
		{"russian text after ampersand", "A &йсмическими B", "A &amp;йсмическими B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(fixUnescapedAmpersands([]byte(tt.input)))
			if result != tt.expected {
				t.Errorf("fixUnescapedAmpersands(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test malformed tag fixing
func TestFixMalformedTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"tag starting with number", "<123abc>", "&lt;123abc>"},
		{"tag with ellipsis", "<...>", "&lt;...>"},
		{"tag with double dash", "<-->", "&lt;-->"},
		{"valid tag", "<p>text</p>", "<p>text</p>"},
		{"tag with hyphen and space", "<- text>", "&lt;- text>"},
		{"tag with hyphen and number", "<-123>", "&lt;-123>"},
		// Cyrillic text after < (invalid XML element name)
		{"cyrillic after <", "<Привет>", "&lt;Привет>"},
		{"cyrillic word in text", "text <образуется more", "text &lt;образуется more"},
		// Space/whitespace after <
		{"space after <", "< space", "&lt; space"},
		{"newline after <", "<\ntext", "&lt;\ntext"},
		// Bare < at end
		{"bare < at end", "text <", "text &lt;"},
		// Valid tags should not be modified
		{"valid closing tag", "</p>", "</p>"},
		{"valid comment", "<!-- comment -->", "<!-- comment -->"},
		{"valid processing instruction", "<?xml?>", "<?xml?>"},
		{"valid tag with underscore", "<_tag>", "<_tag>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(fixMalformedTags([]byte(tt.input)))
			if result != tt.expected {
				t.Errorf("fixMalformedTags(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test complete FB2 parsing with malformed XML
func TestParseFB2MetadataWithMalformedXML(t *testing.T) {
	tests := []struct {
		name        string
		fb2Content  string
		shouldParse bool
		checkTitle  string
	}{
		{
			name: "valid FB2",
			fb2Content: `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author>
        <first-name>Иван</first-name>
        <last-name>Иванов</last-name>
      </author>
      <book-title>Test Book</book-title>
      <lang>ru</lang>
      <sequence name="Test Series" number="5"/>
    </title-info>
  </description>
</FictionBook>`,
			shouldParse: true,
			checkTitle:  "Test Book",
		},
		{
			name: "series with year-month format",
			fb2Content: `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author>
        <first-name>Test</first-name>
        <last-name>Author</last-name>
      </author>
      <book-title>Magazine Issue</book-title>
      <lang>ru</lang>
      <sequence name="Magazine" number="1996 02"/>
    </title-info>
  </description>
</FictionBook>`,
			shouldParse: true,
			checkTitle:  "Magazine Issue",
		},
		{
			name: "series with corrupted data",
			fb2Content: `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author>
        <first-name>Test</first-name>
        <last-name>Author</last-name>
      </author>
      <book-title>Corrupted Series</book-title>
      <lang>ru</lang>
      <sequence name="Series" number="« name=»Рассказы"/>
    </title-info>
  </description>
</FictionBook>`,
			shouldParse: true,
			checkTitle:  "Corrupted Series",
		},
		{
			name: "FB2 with unescaped ampersand",
			fb2Content: `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author>
        <first-name>Test</first-name>
        <last-name>Author</last-name>
      </author>
      <book-title>Book &amp; Title</book-title>
      <annotation>
        <p>Text with & ampersand</p>
      </annotation>
      <lang>en</lang>
    </title-info>
  </description>
</FictionBook>`,
			shouldParse: true,
			checkTitle:  "Book & Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := ParseFB2MetadataFromBytes([]byte(tt.fb2Content))

			if tt.shouldParse {
				if err != nil {
					t.Errorf("Expected successful parse but got error: %v", err)
					return
				}
				if metadata == nil {
					t.Errorf("Expected metadata but got nil")
					return
				}
				if tt.checkTitle != "" && metadata.Title != tt.checkTitle {
					t.Errorf("Expected title %q but got %q", tt.checkTitle, metadata.Title)
				}
			} else {
				if err == nil {
					t.Errorf("Expected parse error but got success")
				}
			}
		})
	}
}

// Test series index extraction
func TestParseFB2MetadataSeriesIndex(t *testing.T) {
	tests := []struct {
		name          string
		seriesNumber  string
		expectedIndex int
	}{
		{"normal number", "5", 5},
		{"year-month", "1996 02", 1996},
		{"corrupted", "« name=»Test", 1},
		{"empty", "", 0},
		{"underscore", "09_2", 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb2Content := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author>
        <first-name>Test</first-name>
        <last-name>Author</last-name>
      </author>
      <book-title>Test</book-title>
      <lang>en</lang>
      <sequence name="Series" number="` + tt.seriesNumber + `"/>
    </title-info>
  </description>
</FictionBook>`

			metadata, err := ParseFB2MetadataFromBytes([]byte(fb2Content))
			if err != nil {
				t.Errorf("Parse failed: %v", err)
				return
			}

			if metadata.SeriesIndex != tt.expectedIndex {
				t.Errorf("Expected series index %d but got %d", tt.expectedIndex, metadata.SeriesIndex)
			}
		})
	}
}
