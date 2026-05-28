package httptransport

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"navigation/internal/domain"
	"navigation/internal/service"
)

type testStore struct {
	sites    []domain.Site
	notes    []domain.Note
	contents map[string]string
	user     domain.User
	settings domain.AppSettings
}

func (s *testStore) ListSites() ([]domain.Site, error) {
	sites := make([]domain.Site, len(s.sites))
	copy(sites, s.sites)
	return sites, nil
}

func (s *testStore) SaveSites(sites []domain.Site) error {
	s.sites = make([]domain.Site, len(sites))
	copy(s.sites, sites)
	return nil
}

func (s *testStore) GetUser() (domain.User, error) {
	if s.user.Username == "" {
		return domain.User{}, sql.ErrNoRows
	}
	return s.user, nil
}

func (s *testStore) SaveUser(user domain.User) error {
	s.user = user
	return nil
}

func (s *testStore) GetSettings() (domain.AppSettings, error) {
	return s.settings, nil
}

func (s *testStore) SaveSettings(settings domain.AppSettings) error {
	s.settings = settings
	return nil
}

func (s *testStore) ListNotes(status, query string) ([]domain.Note, error) {
	notes := []domain.Note{}
	for _, note := range s.notes {
		if status != "" && note.Status != status {
			continue
		}
		notes = append(notes, note)
	}
	return notes, nil
}

func (s *testStore) GetNote(id string) (domain.Note, error) {
	for _, note := range s.notes {
		if note.ID == id {
			return note, nil
		}
	}
	return domain.Note{}, sql.ErrNoRows
}

func (s *testStore) CreateNote(note domain.Note) error {
	s.notes = append(s.notes, note)
	return nil
}

func (s *testStore) UpdateNote(note domain.Note) error {
	for i := range s.notes {
		if s.notes[i].ID == note.ID {
			s.notes[i] = note
			return nil
		}
	}
	return sql.ErrNoRows
}

func (s *testStore) SoftDeleteNote(id, deletedAt string) error {
	for i := range s.notes {
		if s.notes[i].ID == id {
			s.notes[i].Status = domain.NoteStatusDeleted
			s.notes[i].DeletedAt = deletedAt
			s.notes[i].UpdatedAt = deletedAt
			return nil
		}
	}
	return sql.ErrNoRows
}

func (s *testStore) NewRelativePath(id string, now time.Time) string {
	return "notes/2026/05/" + id + ".md"
}

func (s *testStore) Write(relativePath, content string) error {
	if s.contents == nil {
		s.contents = map[string]string{}
	}
	s.contents[relativePath] = content
	return nil
}

func (s *testStore) Read(relativePath string) (string, error) {
	content, ok := s.contents[relativePath]
	if !ok {
		return "", sql.ErrNoRows
	}
	return content, nil
}

func (s *testStore) MoveToTrash(relativePath, id string) (string, error) {
	return relativePath, nil
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	store := &testStore{
		sites: []domain.Site{{ID: "site-1", Name: "Go", URL: "https://go.dev", Category: "Dev", Sort: 1}},
	}
	auth, err := service.NewAuthService(store)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	static := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("index")}}
	return NewHandler(service.NewSiteService(store), auth, service.NewNoteService(store, store), static).Routes()
}

func TestAnonymousReadEndpoints(t *testing.T) {
	handler := newTestHandler(t)
	paths := []string{"/api/sites", "/api/categories", "/api/stats", "/api/settings", "/api/category-stats"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
		})
	}
}

func TestAnonymousWriteEndpointsRequireLogin(t *testing.T) {
	handler := newTestHandler(t)
	requests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/api/sites", body: `{}`},
		{method: http.MethodPut, path: "/api/sites/site-1", body: `{}`},
		{method: http.MethodDelete, path: "/api/sites/site-1"},
		{method: http.MethodPut, path: "/api/categories/Dev", body: `{"name":"Docs"}`},
		{method: http.MethodDelete, path: "/api/categories/Dev"},
		{method: http.MethodPut, path: "/api/settings", body: `{}`},
		{method: http.MethodGet, path: "/api/notes"},
		{method: http.MethodPost, path: "/api/notes", body: `{}`},
		{method: http.MethodGet, path: "/api/notes/note-1"},
		{method: http.MethodPut, path: "/api/notes/note-1", body: `{}`},
		{method: http.MethodDelete, path: "/api/notes/note-1"},
	}

	for _, request := range requests {
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(request.method, request.path, bytes.NewBufferString(request.body)))
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestAuthenticatedNoteLifecycle(t *testing.T) {
	store := &testStore{
		sites:    []domain.Site{{ID: "site-1", Name: "Go", URL: "https://go.dev", Category: "Dev", Sort: 1}},
		contents: map[string]string{},
	}
	auth, err := service.NewAuthService(store)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	token, _, err := auth.Login(service.DefaultUsername, service.DefaultPassword)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	static := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("index")}}
	handler := NewHandler(service.NewSiteService(store), auth, service.NewNoteService(store, store), static).Routes()

	createReq := httptest.NewRequest(http.MethodPost, "/api/notes", bytes.NewBufferString(`{"title":"新笔记","content":"# 新笔记\n\n正文","tags":[" idea ","idea"],"pinned":true}`))
	createReq.AddCookie(&http.Cookie{Name: "navigation_session", Value: token})
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d, body = %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	var created domain.NoteContent
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode created note: %v", err)
	}
	if created.ID == "" || created.FilePath == "" {
		t.Fatalf("created note missing id or path: %#v", created)
	}
	if len(created.Tags) != 1 || created.Tags[0] != "idea" {
		t.Fatalf("tags = %#v, want normalized idea", created.Tags)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/notes/"+created.ID, nil)
	getReq.AddCookie(&http.Cookie{Name: "navigation_session", Value: token})
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}
}

func TestCreateInvalidNoteReturnsBadRequest(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(`{"username":"admin","password":"admin"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", rec.Code, http.StatusOK)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/notes", bytes.NewBufferString(`{"content":"正文"}`))
	for _, cookie := range rec.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
