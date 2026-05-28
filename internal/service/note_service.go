package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"navigation/internal/domain"
)

const (
	maxNoteTitleLength   = 120
	maxNoteContentLength = 1024 * 1024
	maxNoteTags          = 20
	maxNoteTagLength     = 32
	maxNoteSummaryLength = 120
)

// NoteMetaStore 定义笔记元数据持久化能力。
type NoteMetaStore interface {
	ListNotes(status, query string) ([]domain.Note, error)
	GetNote(id string) (domain.Note, error)
	CreateNote(domain.Note) error
	UpdateNote(domain.Note) error
	SoftDeleteNote(id, deletedAt string) error
}

// NoteContentStore 定义笔记 Markdown 文件读写能力。
type NoteContentStore interface {
	NewRelativePath(id string, now time.Time) string
	Write(relativePath, content string) error
	Read(relativePath string) (string, error)
	MoveToTrash(relativePath, id string) (string, error)
}

// NoteService 封装笔记业务规则。
type NoteService struct {
	mu      sync.Mutex
	meta    NoteMetaStore
	content NoteContentStore
}

// NewNoteService 创建笔记服务。
func NewNoteService(meta NoteMetaStore, content NoteContentStore) *NoteService {
	return &NoteService{meta: meta, content: content}
}

// ListNotes 返回笔记列表，不读取 Markdown 全文。
func (s *NoteService) ListNotes(status, query string) ([]domain.Note, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		status = domain.NoteStatusActive
	}
	if !validNoteListStatus(status) {
		return nil, ValidationError{Message: "笔记状态不正确"}
	}
	notes, err := s.meta.ListNotes(status, strings.TrimSpace(query))
	if err != nil {
		return nil, StoreError{Op: StoreOpRead, Err: err}
	}
	return notes, nil
}

// GetNote 返回笔记元数据和 Markdown 正文。
func (s *NoteService) GetNote(id string) (domain.NoteContent, error) {
	note, err := s.meta.GetNote(strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NoteContent{}, ErrNotFound
		}
		return domain.NoteContent{}, StoreError{Op: StoreOpRead, Err: err}
	}
	content, err := s.content.Read(note.FilePath)
	if err != nil {
		return domain.NoteContent{}, StoreError{Op: StoreOpRead, Err: err}
	}
	return domain.NoteContent{Note: note, Content: content}, nil
}

// CreateNote 创建笔记文件和元数据。
func (s *NoteService) CreateNote(input domain.NoteContent) (domain.NoteContent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nowTime := time.Now()
	now := nowTime.Format(time.RFC3339)
	input.ID = newNoteID()
	input.CreatedAt = now
	input.UpdatedAt = now
	input.DeletedAt = ""
	input.FilePath = s.content.NewRelativePath(input.ID, nowTime)
	if input.Status == "" {
		input.Status = domain.NoteStatusActive
	}
	normalizeNoteContent(&input)
	if err := validateNoteContent(input); err != nil {
		return domain.NoteContent{}, err
	}

	if err := s.content.Write(input.FilePath, input.Content); err != nil {
		return domain.NoteContent{}, StoreError{Op: StoreOpSave, Err: err}
	}
	if err := s.meta.CreateNote(input.Note); err != nil {
		return domain.NoteContent{}, StoreError{Op: StoreOpSave, Err: err}
	}
	return input, nil
}

// UpdateNote 更新笔记文件和元数据。
func (s *NoteService) UpdateNote(id string, input domain.NoteContent) (domain.NoteContent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.meta.GetNote(strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NoteContent{}, ErrNotFound
		}
		return domain.NoteContent{}, StoreError{Op: StoreOpRead, Err: err}
	}
	if existing.Status == domain.NoteStatusDeleted {
		return domain.NoteContent{}, ErrNotFound
	}

	input.ID = existing.ID
	input.FilePath = existing.FilePath
	input.CreatedAt = existing.CreatedAt
	input.DeletedAt = existing.DeletedAt
	input.UpdatedAt = time.Now().Format(time.RFC3339)
	if input.Status == "" {
		input.Status = existing.Status
	}
	normalizeNoteContent(&input)
	if err := validateNoteContent(input); err != nil {
		return domain.NoteContent{}, err
	}

	if err := s.content.Write(input.FilePath, input.Content); err != nil {
		return domain.NoteContent{}, StoreError{Op: StoreOpSave, Err: err}
	}
	if err := s.meta.UpdateNote(input.Note); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NoteContent{}, ErrNotFound
		}
		return domain.NoteContent{}, StoreError{Op: StoreOpSave, Err: err}
	}
	return input, nil
}

// DeleteNote 将笔记软删除，不物理删除 Markdown 文件。
func (s *NoteService) DeleteNote(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if _, err := s.meta.GetNote(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return StoreError{Op: StoreOpRead, Err: err}
	}
	if err := s.meta.SoftDeleteNote(id, time.Now().Format(time.RFC3339)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return StoreError{Op: StoreOpSave, Err: err}
	}
	return nil
}

func normalizeNoteContent(input *domain.NoteContent) {
	input.Title = strings.TrimSpace(input.Title)
	input.Status = strings.TrimSpace(input.Status)
	input.Tags = normalizeTags(input.Tags)
	if input.Title == "" {
		input.Title = titleFromMarkdown(input.Content)
	}
	input.Summary = makeNoteSummary(input.Content)
}

func validateNoteContent(input domain.NoteContent) error {
	if input.Title == "" {
		return ValidationError{Message: "笔记标题不能为空"}
	}
	if utf8.RuneCountInString(input.Title) > maxNoteTitleLength {
		return ValidationError{Message: "笔记标题不能超过 120 个字符"}
	}
	if len([]byte(input.Content)) > maxNoteContentLength {
		return ValidationError{Message: "笔记正文不能超过 1MB"}
	}
	if !validNoteWriteStatus(input.Status) {
		return ValidationError{Message: "笔记状态不正确"}
	}
	return nil
}

func validNoteWriteStatus(status string) bool {
	return status == domain.NoteStatusActive || status == domain.NoteStatusArchived
}

func validNoteListStatus(status string) bool {
	return validNoteWriteStatus(status) || status == domain.NoteStatusDeleted
}

func normalizeTags(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		if utf8.RuneCountInString(tag) > maxNoteTagLength {
			tag = string([]rune(tag)[:maxNoteTagLength])
		}
		seen[tag] = true
		normalized = append(normalized, tag)
		if len(normalized) == maxNoteTags {
			break
		}
	}
	return normalized
}

func titleFromMarkdown(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

var markdownTokenPattern = regexp.MustCompile(`[#>*_` + "`" + `\[\]()]`)

func makeNoteSummary(content string) string {
	text := markdownTokenPattern.ReplaceAllString(content, " ")
	text = strings.Join(strings.Fields(text), " ")
	runes := []rune(text)
	if len(runes) > maxNoteSummaryLength {
		return string(runes[:maxNoteSummaryLength])
	}
	return text
}

func newNoteID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "note_" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("note_%d", time.Now().UnixNano())
}
