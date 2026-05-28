package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NoteFileStore 管理笔记 Markdown 文件读写。
type NoteFileStore struct {
	dataDir  string
	notesDir string
}

// NewNoteFileStore 创建笔记目录并返回文件存储。
func NewNoteFileStore(dataDir string) (*NoteFileStore, error) {
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	notesDir := filepath.Join(absDataDir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return nil, err
	}
	return &NoteFileStore{dataDir: absDataDir, notesDir: notesDir}, nil
}

// NewRelativePath 根据时间和笔记 ID 生成相对 dataDir 的 Markdown 路径。
func (s *NoteFileStore) NewRelativePath(id string, now time.Time) string {
	return filepath.ToSlash(filepath.Join("notes", now.Format("2006"), now.Format("01"), fmt.Sprintf("%s.md", id)))
}

// Write 写入 Markdown 正文。
func (s *NoteFileStore) Write(relativePath, content string) error {
	target, err := s.resolve(relativePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(content), 0644)
}

// Read 读取 Markdown 正文。
func (s *NoteFileStore) Read(relativePath string) (string, error) {
	target, err := s.resolve(relativePath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MoveToTrash 将笔记文件移动到 notes/.trash/ 下。
func (s *NoteFileStore) MoveToTrash(relativePath, id string) (string, error) {
	source, err := s.resolve(relativePath)
	if err != nil {
		return "", err
	}
	trashRelative := filepath.ToSlash(filepath.Join("notes", ".trash", fmt.Sprintf("%s.md", id)))
	target, err := s.resolve(trashRelative)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", err
	}
	if err := os.Rename(source, target); err != nil {
		return "", err
	}
	return trashRelative, nil
}

func (s *NoteFileStore) resolve(relativePath string) (string, error) {
	relativePath = strings.TrimSpace(relativePath)
	if relativePath == "" {
		return "", fmt.Errorf("笔记路径不能为空")
	}
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("笔记路径不能是绝对路径")
	}

	cleaned := filepath.Clean(filepath.FromSlash(relativePath))
	if cleaned == "." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == ".." {
		return "", fmt.Errorf("笔记路径不能包含上级目录")
	}
	if cleaned != "notes" && !strings.HasPrefix(cleaned, "notes"+string(filepath.Separator)) {
		return "", fmt.Errorf("笔记路径必须位于 notes 目录")
	}

	target := filepath.Join(s.dataDir, cleaned)
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(s.notesDir, absTarget)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("笔记路径越界")
	}
	return absTarget, nil
}
