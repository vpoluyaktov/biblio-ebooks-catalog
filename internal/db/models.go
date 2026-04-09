package db

import "time"

type Library struct {
	ID              int64     `db:"id" json:"id"`
	Name            string    `db:"name" json:"name"`
	Path            string    `db:"path" json:"path"`
	InpxPath        string    `db:"inpx" json:"inpx_path"`
	Version         string    `db:"version" json:"version"`
	FirstAuthorOnly bool      `db:"first_author" json:"first_author_only"`
	WithoutDeleted  bool      `db:"without_deleted" json:"without_deleted"`
	Enabled         bool      `db:"enabled" json:"enabled"`
	LangFilter      string    `db:"lang_filter" json:"lang_filter"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

type Book struct {
	ID            int64     `db:"id" json:"id"`
	LibraryID     int64     `db:"library_id" json:"library_id"`
	Title         string    `db:"title" json:"title"`
	Lang          string    `db:"lang" json:"lang"`
	File          string    `db:"file" json:"file"`
	Archive       string    `db:"archive" json:"archive"`
	Format        string    `db:"format" json:"format"`
	Size          int64     `db:"size" json:"size"`
	Rating        int       `db:"rating" json:"rating"`
	Deleted       bool      `db:"deleted" json:"deleted"`
	AddedAt       time.Time `db:"added_at" json:"added_at"`
	IDInLib       int64     `db:"id_in_lib" json:"id_in_lib"`
	FirstAuthorID *int64    `db:"first_author_id" json:"first_author_id"`
	Keywords      string    `db:"keywords" json:"keywords"`
}

type Author struct {
	ID         int64  `db:"id" json:"id"`
	LibraryID  int64  `db:"library_id" json:"library_id"`
	LastName   string `db:"last_name" json:"last_name"`
	FirstName  string `db:"first_name" json:"first_name"`
	MiddleName string `db:"middle_name" json:"middle_name"`
}

func (a Author) FullName() string {
	name := a.LastName
	if a.FirstName != "" {
		if name != "" {
			name += " "
		}
		name += a.FirstName
	}
	if a.MiddleName != "" {
		if name != "" {
			name += " "
		}
		name += a.MiddleName
	}
	return name
}

type Series struct {
	ID        int64  `db:"id" json:"id"`
	LibraryID int64  `db:"library_id" json:"library_id"`
	Name      string `db:"name" json:"name"`
}

type Genre struct {
	ID       int    `db:"id" json:"id"`
	ParentID int    `db:"parent_id" json:"parent_id"`
	Name     string `db:"name" json:"name"`
	Code     string `db:"code" json:"code"`
}

type BookAuthor struct {
	BookID   int64 `db:"book_id"`
	AuthorID int64 `db:"author_id"`
}

type BookSeries struct {
	BookID   int64 `db:"book_id"`
	SeriesID int64 `db:"series_id"`
	SeqNum   int   `db:"seq_num"`
}

type BookGenre struct {
	BookID  int64 `db:"book_id"`
	GenreID int   `db:"genre_id"`
}

// Extended types for queries with joins

type BookWithDetails struct {
	Book
	AuthorName string `db:"author_name"`
	SeriesName string `db:"series_name"`
	SeqNum     int    `db:"seq_num"`
}

type AuthorWithCount struct {
	Author
	BookCount int `db:"book_count"`
}

type SeriesWithCount struct {
	Series
	BookCount int `db:"book_count"`
}

type GenreWithCount struct {
	Genre
	BookCount int `db:"book_count"`
}

// User roles
const (
	RoleAdmin    = "admin"
	RoleReadonly = "readonly"
)

type User struct {
	ID           int64     `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         string    `db:"role" json:"role"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

type Session struct {
	ID        string    `db:"id"`
	UserID    int64     `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// OIDCSession stores OAuth2/OIDC tokens server-side
type OIDCSession struct {
	ID           string    `db:"id"`
	Username     string    `db:"username"`
	Role         string    `db:"role"`
	IDToken      string    `db:"id_token"`
	AccessToken  string    `db:"access_token"`
	RefreshToken string    `db:"refresh_token"`
	ExpiresAt    time.Time `db:"expires_at"`
	CreatedAt    time.Time `db:"created_at"`
}
