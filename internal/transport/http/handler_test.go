package httptransport

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"navigation/internal/domain"
	"navigation/internal/service"
)

type testStore struct {
	sites    []domain.Site
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

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	store := &testStore{
		sites: []domain.Site{{ID: "site-1", Name: "Go", URL: "https://go.dev", Category: "Dev", Sort: 1}},
	}
	auth, err := service.NewAuthService(store)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	static := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("index")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
	}
	return NewHandler(service.NewSiteService(store), auth, static).Routes()
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

func TestStaticRoutes(t *testing.T) {
	handler := newTestHandler(t)
	requests := []struct {
		path string
		want int
	}{
		{path: "/", want: http.StatusOK},
		{path: "/index.html", want: http.StatusOK},
		{path: "/assets/app.js", want: http.StatusOK},
		{path: "/assets/missing.js", want: http.StatusNotFound},
	}

	for _, request := range requests {
		t.Run(request.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, request.path, nil))
			if recorder.Code != request.want {
				t.Fatalf("status = %d, want %d", recorder.Code, request.want)
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
