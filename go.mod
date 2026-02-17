module biblio-ebooks-catalog

go 1.24.0

replace github.com/vpoluyaktov/biblio-ebook-parser => ../biblio-ebook-parser

toolchain go1.24.12

require (
	github.com/fogleman/gg v1.3.0
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/jmoiron/sqlx v1.4.0
	github.com/mattn/go-sqlite3 v1.14.33
	github.com/vpoluyaktov/biblio-ebook-parser v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.47.0
	golang.org/x/text v0.34.0
	gopkg.in/yaml.v3 v3.0.1
)

require golang.org/x/image v0.35.0 // indirect
