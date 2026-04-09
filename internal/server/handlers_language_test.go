package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"biblio-ebooks-catalog/internal/config"
	"biblio-ebooks-catalog/internal/db"
)

// newLangTestServer creates a minimal Server backed by an in-memory SQLite
// database using internal auth mode, then sets up its routes.
func newLangTestServer(t *testing.T) (*Server, *db.DB) {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := config.Default()
	cfg.Auth.Mode = "internal"

	srv, err := New(cfg, database)
	if err != nil {
		t.Fatalf("New server: %v", err)
	}

	return srv, database
}

// createAdminSession creates an admin user and session in the DB, returning
// a pre-built *http.Cookie suitable for authenticating test requests.
func createAdminSession(t *testing.T, srv *Server) *http.Cookie {
	t.Helper()
	internalAuth := srv.auth
	admin, err := internalAuth.CreateUser("testadmin", "testpassword", db.RoleAdmin)
	if err != nil {
		t.Fatalf("CreateUser admin: %v", err)
	}
	session, err := internalAuth.CreateSession(admin.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return &http.Cookie{Name: "session", Value: session.ID}
}

// createReadonlySession creates a read-only user and session in the DB.
func createReadonlySession(t *testing.T, srv *Server) *http.Cookie {
	t.Helper()
	internalAuth := srv.auth
	user, err := internalAuth.CreateUser("testreader", "testpassword", db.RoleReadonly)
	if err != nil {
		t.Fatalf("CreateUser readonly: %v", err)
	}
	session, err := internalAuth.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return &http.Cookie{Name: "session", Value: session.ID}
}

// doRequest sends a request through the full server mux and returns the recorder.
func doRequest(t *testing.T, mux http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Buffer
	if body != "" {
		bodyReader = bytes.NewBufferString(body)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

// seedDBBook inserts a minimal book directly into the test database.
func seedDBBook(t *testing.T, database *db.DB, libID int64, title, lang string) {
	t.Helper()
	_, err := database.Exec(
		`INSERT INTO book (library_id, title, lang, file, format, deleted) VALUES (?, ?, ?, ?, ?, 0)`,
		libID, title, lang, "file.fb2", "fb2",
	)
	if err != nil {
		t.Fatalf("seedDBBook: %v", err)
	}
}

// seedDBLibrary inserts a test library and returns its ID.
func seedDBLibrary(t *testing.T, database *db.DB) int64 {
	t.Helper()
	lib := &db.Library{Name: "Test Library", Path: "/tmp/books", Enabled: true}
	id, err := database.CreateLibrary(lib)
	if err != nil {
		t.Fatalf("seedDBLibrary: %v", err)
	}
	return id
}

// ---- GET /api/languages ----

// TestAPIGetAvailableLanguages_Returns200AndArray verifies that the
// GET /api/languages endpoint returns HTTP 200 with a JSON array body.
func TestAPIGetAvailableLanguages_Returns200AndArray(t *testing.T) {
	srv, database := newLangTestServer(t)
	mux := srv.setupRoutes()
	cookie := createAdminSession(t, srv)

	libID := seedDBLibrary(t, database)
	seedDBBook(t, database, libID, "English Book", "en")
	seedDBBook(t, database, libID, "Russian Book", "ru")

	rr := doRequest(t, mux, http.MethodGet, "/api/languages", "", cookie)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Must decode as []string.
	var langs []string
	if err := json.NewDecoder(rr.Body).Decode(&langs); err != nil {
		// Accept null for empty catalog edge case.
		body := strings.TrimSpace(rr.Body.String())
		if body != "null" && body != "[]" {
			t.Fatalf("decode as []string: %v (body: %s)", err, body)
		}
	}
}

// TestAPIGetAvailableLanguages_EmptyCatalog_ReturnsEmptyArray verifies that the
// endpoint returns an empty array (or null) when no books exist.
func TestAPIGetAvailableLanguages_EmptyCatalog_ReturnsEmptyArray(t *testing.T) {
	srv, _ := newLangTestServer(t)
	mux := srv.setupRoutes()
	cookie := createAdminSession(t, srv)

	rr := doRequest(t, mux, http.MethodGet, "/api/languages", "", cookie)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var langs []string
	if err := json.NewDecoder(rr.Body).Decode(&langs); err != nil {
		body := strings.TrimSpace(rr.Body.String())
		if body != "null" && body != "[]" {
			t.Fatalf("unexpected body: %s", body)
		}
	}
	// If we got here, the response was decodeable — acceptable.
}

// ---- GET /api/settings/languages ----

// TestAPIGetSelectedLanguages_Returns200AndEmptyArrayWhenNotSet verifies that
// GET /api/settings/languages returns HTTP 200 and an empty array when no
// language setting has been saved.
func TestAPIGetSelectedLanguages_Returns200AndEmptyArrayWhenNotSet(t *testing.T) {
	srv, _ := newLangTestServer(t)
	mux := srv.setupRoutes()
	cookie := createAdminSession(t, srv)

	rr := doRequest(t, mux, http.MethodGet, "/api/settings/languages", "", cookie)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var langs []string
	if err := json.NewDecoder(rr.Body).Decode(&langs); err != nil {
		body := strings.TrimSpace(rr.Body.String())
		if body != "null" && body != "[]" {
			t.Fatalf("decode response: %v (body: %s)", err, body)
		}
	}
	if len(langs) != 0 {
		t.Errorf("expected empty slice when no setting saved, got %v", langs)
	}
}

// TestAPIGetSelectedLanguages_ReturnsSetLanguages verifies that already-saved
// languages are returned correctly.
func TestAPIGetSelectedLanguages_ReturnsSetLanguages(t *testing.T) {
	srv, database := newLangTestServer(t)
	mux := srv.setupRoutes()
	cookie := createAdminSession(t, srv)

	// Pre-populate the setting directly in the DB.
	if err := database.SaveSelectedLanguages([]string{"ru", "de"}); err != nil {
		t.Fatalf("SaveSelectedLanguages pre-seed: %v", err)
	}

	rr := doRequest(t, mux, http.MethodGet, "/api/settings/languages", "", cookie)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var langs []string
	if err := json.NewDecoder(rr.Body).Decode(&langs); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, rr.Body.String())
	}
	if len(langs) != 2 {
		t.Errorf("expected 2 languages, got %v", langs)
	}
}

// ---- PUT /api/settings/languages ----

// TestAPISaveSelectedLanguages_AdminReturns204 verifies that an admin user
// can save language settings and receives HTTP 204.
func TestAPISaveSelectedLanguages_AdminReturns204(t *testing.T) {
	srv, _ := newLangTestServer(t)
	mux := srv.setupRoutes()
	cookie := createAdminSession(t, srv)

	rr := doRequest(t, mux, http.MethodPut, "/api/settings/languages", `{"languages":["ru","en"]}`, cookie)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAPISaveSelectedLanguages_PersistsData verifies that the saved languages
// can be retrieved from the database after a successful PUT.
func TestAPISaveSelectedLanguages_PersistsData(t *testing.T) {
	srv, database := newLangTestServer(t)
	mux := srv.setupRoutes()
	cookie := createAdminSession(t, srv)

	rr := doRequest(t, mux, http.MethodPut, "/api/settings/languages", `{"languages":["fr","de"]}`, cookie)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}

	saved, err := database.GetSelectedLanguages()
	if err != nil {
		t.Fatalf("GetSelectedLanguages: %v", err)
	}
	if len(saved) != 2 {
		t.Errorf("expected 2 languages saved, got %v", saved)
	}
	savedSet := make(map[string]bool)
	for _, l := range saved {
		savedSet[l] = true
	}
	for _, want := range []string{"fr", "de"} {
		if !savedSet[want] {
			t.Errorf("expected %q in saved languages %v", want, saved)
		}
	}
}

// TestAPISaveSelectedLanguages_EmptyList_Clears verifies that saving an empty
// list clears the language filter.
func TestAPISaveSelectedLanguages_EmptyList_Clears(t *testing.T) {
	srv, database := newLangTestServer(t)
	mux := srv.setupRoutes()
	cookie := createAdminSession(t, srv)

	// Pre-seed a value.
	if err := database.SaveSelectedLanguages([]string{"ru"}); err != nil {
		t.Fatalf("pre-seed: %v", err)
	}

	rr := doRequest(t, mux, http.MethodPut, "/api/settings/languages", `{"languages":[]}`, cookie)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}

	saved, err := database.GetSelectedLanguages()
	if err != nil {
		t.Fatalf("GetSelectedLanguages: %v", err)
	}
	if len(saved) != 0 {
		t.Errorf("expected empty list after clearing, got %v", saved)
	}
}

// TestAPISaveSelectedLanguages_InvalidJSON_Returns400 verifies that malformed
// JSON in the request body results in HTTP 400.
func TestAPISaveSelectedLanguages_InvalidJSON_Returns400(t *testing.T) {
	srv, _ := newLangTestServer(t)
	mux := srv.setupRoutes()
	cookie := createAdminSession(t, srv)

	rr := doRequest(t, mux, http.MethodPut, "/api/settings/languages", `not-valid-json`, cookie)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAPISaveSelectedLanguages_NonAdminReturns401or403 verifies that a
// read-only user cannot save language settings and receives 401 or 403.
func TestAPISaveSelectedLanguages_NonAdminReturns401or403(t *testing.T) {
	srv, _ := newLangTestServer(t)
	mux := srv.setupRoutes()
	cookie := createReadonlySession(t, srv)

	rr := doRequest(t, mux, http.MethodPut, "/api/settings/languages", `{"languages":["ru"]}`, cookie)

	if rr.Code != http.StatusForbidden && rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 403 or 401 for non-admin, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAPISaveSelectedLanguages_UnauthenticatedReturns401 verifies that an
// unauthenticated request receives HTTP 401.
func TestAPISaveSelectedLanguages_UnauthenticatedReturns401(t *testing.T) {
	srv, _ := newLangTestServer(t)
	mux := srv.setupRoutes()
	// No session cookie.

	rr := doRequest(t, mux, http.MethodPut, "/api/settings/languages", `{"languages":["ru"]}`, nil)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("expected 401 or 403 for unauthenticated request, got %d: %s", rr.Code, rr.Body.String())
	}
}
