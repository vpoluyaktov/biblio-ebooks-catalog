package bookfile

import (
	"bytes"
	"image/jpeg"
	"testing"
)

func TestGeneratePlaceholderCover(t *testing.T) {
	tests := []struct {
		name   string
		title  string
		author string
	}{
		{
			name:   "English title and author",
			title:  "The Great Adventure",
			author: "John Smith",
		},
		{
			name:   "Cyrillic title and author",
			title:  "Война и мир",
			author: "Лев Толстой",
		},
		{
			name:   "Long title",
			title:  "A Very Long Book Title That Should Wrap Across Multiple Lines",
			author: "Author Name",
		},
		{
			name:   "Empty author",
			title:  "Book Without Author",
			author: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := GeneratePlaceholderCover(tt.title, tt.author)
			if err != nil {
				t.Fatalf("GeneratePlaceholderCover() error = %v", err)
			}

			if len(data) == 0 {
				t.Fatal("GeneratePlaceholderCover() returned empty data")
			}

			// Verify it's a valid JPEG
			_, err = jpeg.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("Generated data is not valid JPEG: %v", err)
			}
		})
	}
}
