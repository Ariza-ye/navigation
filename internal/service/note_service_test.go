package service

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"navigation/internal/domain"
)

type memoryNoteStore struct {
	notes []domain.Note
	files map[string]string
}

func (s *memoryNoteStore) ListNotes(status, query string) ([]domain.Note, error) {
	notes := []domain.Note{}
	for _, note := range s.notes {
		if status == "" || note.Status == status {
			notes = append(notes, note)
		}
	}
	return notes, nil
}

func (s *memoryNoteStore) GetNote(id string) (domain.Note, error) {
	for _, note := range s.notes {
		if note.ID == id {
			return note, nil
		}
	}
	return domain.Note{}, sql.ErrNoRows
}

func (s *memoryNoteStore) CreateNote(note domain.Note) error {
	s.notes = append(s.notes, note)
	return nil
}

func (s *memoryNoteStore) UpdateNote(note domain.Note) error {
	for i := range s.notes {
		if s.notes[i].ID == note.ID {
			s.notes[i] = note
			return nil
		}
	}
	return sql.ErrNoRows
}

func (s *memoryNoteStore) SoftDeleteNote(id, deletedAt string) error {
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

func (s *memoryNoteStore) NewRelativePath(id string, now time.Time) string {
	return "notes/2026/05/" + id + ".md"
}

func (s *memoryNoteStore) Write(relativePath, content string) error {
	if s.files == nil {
		s.files = map[string]string{}
	}
	s.files[relativePath] = content
	return nil
}

func (s *memoryNoteStore) Read(relativePath string) (string, error) {
	content, ok := s.files[relativePath]
	if !ok {
		return "", sql.ErrNoRows
	}
	return content, nil
}

func (s *memoryNoteStore) MoveToTrash(relativePath, id string) (string, error) {
	return relativePath, nil
}

func TestCreateNoteCompletesMetadataAndSummary(t *testing.T) {
	store := &memoryNoteStore{files: map[string]string{}}
	service := NewNoteService(store, store)

	note, err := service.CreateNote(domain.NoteContent{
		Note:    domain.Note{Title: "  计划  ", Tags: []string{" idea ", "idea", ""}},
		Content: "# 计划\n\n正文内容",
	})
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}
	if note.ID == "" || note.CreatedAt == "" || note.UpdatedAt == "" || note.FilePath == "" {
		t.Fatalf("note metadata incomplete: %#v", note)
	}
	if note.Status != domain.NoteStatusActive {
		t.Fatalf("status = %q, want active", note.Status)
	}
	if note.Summary == "" {
		t.Fatal("summary was not generated")
	}
	if len(note.Tags) != 1 || note.Tags[0] != "idea" {
		t.Fatalf("tags = %#v, want normalized single tag", note.Tags)
	}
	if store.files[note.FilePath] != "# 计划\n\n正文内容" {
		t.Fatal("content was not written")
	}
}

func TestCreateNoteRejectsMissingTitle(t *testing.T) {
	store := &memoryNoteStore{}
	service := NewNoteService(store, store)

	_, err := service.CreateNote(domain.NoteContent{Content: "正文"})
	if err == nil {
		t.Fatal("CreateNote() error = nil, want validation error")
	}
	var validationErr ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %T, want ValidationError", err)
	}
}

func TestGetMissingNoteReturnsNotFound(t *testing.T) {
	store := &memoryNoteStore{}
	service := NewNoteService(store, store)

	_, err := service.GetNote("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetNote() error = %v, want ErrNotFound", err)
	}
}

func TestUpdateNotePreservesIdentityAndCreatedAt(t *testing.T) {
	store := &memoryNoteStore{
		notes: []domain.Note{{
			ID:        "note_1",
			Title:     "旧标题",
			FilePath:  "notes/2026/05/note_1.md",
			Status:    domain.NoteStatusActive,
			CreatedAt: "2026-05-01T00:00:00Z",
			UpdatedAt: "2026-05-01T00:00:00Z",
		}},
		files: map[string]string{"notes/2026/05/note_1.md": "旧内容"},
	}
	service := NewNoteService(store, store)

	note, err := service.UpdateNote("note_1", domain.NoteContent{
		Note:    domain.Note{Title: "新标题", Status: domain.NoteStatusArchived},
		Content: "新内容",
	})
	if err != nil {
		t.Fatalf("UpdateNote() error = %v", err)
	}
	if note.ID != "note_1" || note.CreatedAt != "2026-05-01T00:00:00Z" {
		t.Fatalf("identity changed: %#v", note)
	}
	if note.UpdatedAt == "2026-05-01T00:00:00Z" {
		t.Fatal("UpdatedAt was not refreshed")
	}
	if store.files[note.FilePath] != "新内容" {
		t.Fatal("content was not updated")
	}
}

func TestDeleteNoteSoftDeletes(t *testing.T) {
	store := &memoryNoteStore{notes: []domain.Note{{ID: "note_1", Status: domain.NoteStatusActive}}}
	service := NewNoteService(store, store)

	if err := service.DeleteNote("note_1"); err != nil {
		t.Fatalf("DeleteNote() error = %v", err)
	}
	if store.notes[0].Status != domain.NoteStatusDeleted || store.notes[0].DeletedAt == "" {
		t.Fatalf("note was not soft deleted: %#v", store.notes[0])
	}
}
