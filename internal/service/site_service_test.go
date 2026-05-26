package service

import (
	"errors"
	"testing"

	"navigation/internal/domain"
)

type memorySiteStore struct {
	sites []domain.Site
}

func (s *memorySiteStore) ListSites() ([]domain.Site, error) {
	sites := make([]domain.Site, len(s.sites))
	copy(sites, s.sites)
	return sites, nil
}

func (s *memorySiteStore) SaveSites(sites []domain.Site) error {
	s.sites = make([]domain.Site, len(sites))
	copy(s.sites, sites)
	return nil
}

func TestRenameCategoryUpdatesSites(t *testing.T) {
	store := &memorySiteStore{
		sites: []domain.Site{
			{ID: "site-1", Category: "Dev"},
			{ID: "site-2", Category: "Dev"},
			{ID: "site-3", Category: "Ops"},
		},
	}
	service := NewSiteService(store)

	updated, err := service.RenameCategory("Dev", "Docs")
	if err != nil {
		t.Fatalf("RenameCategory() error = %v", err)
	}
	if updated != 2 {
		t.Fatalf("updated = %d, want %d", updated, 2)
	}

	for _, site := range store.sites[:2] {
		if site.Category != "Docs" {
			t.Fatalf("site category = %q, want %q", site.Category, "Docs")
		}
		if site.UpdatedAt == "" {
			t.Fatal("UpdatedAt was not set")
		}
	}
	if store.sites[2].Category != "Ops" {
		t.Fatalf("unrelated category = %q, want %q", store.sites[2].Category, "Ops")
	}
}

func TestRenameCategoryRejectsExistingTarget(t *testing.T) {
	store := &memorySiteStore{
		sites: []domain.Site{
			{ID: "site-1", Category: "Dev"},
			{ID: "site-2", Category: "Ops"},
		},
	}
	service := NewSiteService(store)

	_, err := service.RenameCategory("Dev", "Ops")
	if err == nil {
		t.Fatal("RenameCategory() error = nil, want validation error")
	}

	var validationErr ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %T, want ValidationError", err)
	}
	if store.sites[0].Category != "Dev" {
		t.Fatalf("site category = %q, want unchanged Dev", store.sites[0].Category)
	}
}
