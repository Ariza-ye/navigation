package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNoteFileStoreWriteRead(t *testing.T) {
	store, err := NewNoteFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewNoteFileStore() error = %v", err)
	}
	path := "notes/2026/05/note_1.md"
	if err := store.Write(path, "# 标题"); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	content, err := store.Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if content != "# 标题" {
		t.Fatalf("content = %q, want %q", content, "# 标题")
	}
}

func TestNoteFileStoreRejectsUnsafePaths(t *testing.T) {
	store, err := NewNoteFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewNoteFileStore() error = %v", err)
	}
	paths := []string{
		filepath.Join(string(filepath.Separator), "tmp", "note.md"),
		"../note.md",
		"notes/../../note.md",
		"other/note.md",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			if err := store.Write(path, "content"); err == nil {
				t.Fatal("Write() error = nil, want unsafe path error")
			}
		})
	}
}

func TestNoteFileStoreNewRelativePath(t *testing.T) {
	store, err := NewNoteFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewNoteFileStore() error = %v", err)
	}
	path := store.NewRelativePath("note_abc", time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC))
	if path != "notes/2026/05/note_abc.md" {
		t.Fatalf("path = %q, want notes/2026/05/note_abc.md", path)
	}
}
