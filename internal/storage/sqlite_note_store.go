package storage

import (
	"database/sql"
	"encoding/json"
	"strings"

	"navigation/internal/domain"
)

// ListNotes 按状态和关键字读取笔记元数据。
func (s *SQLiteSiteStore) ListNotes(status, query string) ([]domain.Note, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		status = domain.NoteStatusActive
	}
	query = strings.TrimSpace(query)

	sqlQuery := `
		SELECT id, title, file_path, summary, tags, status, pinned, created_at, updated_at, deleted_at
		FROM notes
		WHERE status = ?
	`
	args := []any{status}
	if query != "" {
		sqlQuery += " AND (title LIKE ? OR summary LIKE ?)"
		like := "%" + query + "%"
		args = append(args, like, like)
	}
	sqlQuery += " ORDER BY pinned DESC, updated_at DESC"

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []domain.Note{}
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

// GetNote 读取单条笔记元数据。
func (s *SQLiteSiteStore) GetNote(id string) (domain.Note, error) {
	row := s.db.QueryRow(`
		SELECT id, title, file_path, summary, tags, status, pinned, created_at, updated_at, deleted_at
		FROM notes
		WHERE id = ?
	`, id)
	return scanNote(row)
}

// CreateNote 新增笔记元数据。
func (s *SQLiteSiteStore) CreateNote(note domain.Note) error {
	tags, err := json.Marshal(note.Tags)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO notes (
			id, title, file_path, summary, tags, status, pinned, created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, note.ID, note.Title, note.FilePath, note.Summary, string(tags), note.Status, boolToInt(note.Pinned), note.CreatedAt, note.UpdatedAt, note.DeletedAt)
	return err
}

// UpdateNote 更新笔记元数据。
func (s *SQLiteSiteStore) UpdateNote(note domain.Note) error {
	tags, err := json.Marshal(note.Tags)
	if err != nil {
		return err
	}
	result, err := s.db.Exec(`
		UPDATE notes
		SET title = ?, file_path = ?, summary = ?, tags = ?, status = ?, pinned = ?, created_at = ?, updated_at = ?, deleted_at = ?
		WHERE id = ?
	`, note.Title, note.FilePath, note.Summary, string(tags), note.Status, boolToInt(note.Pinned), note.CreatedAt, note.UpdatedAt, note.DeletedAt, note.ID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SoftDeleteNote 将笔记标记为已删除。
func (s *SQLiteSiteStore) SoftDeleteNote(id, deletedAt string) error {
	result, err := s.db.Exec(`
		UPDATE notes
		SET status = ?, deleted_at = ?, updated_at = ?
		WHERE id = ?
	`, domain.NoteStatusDeleted, deletedAt, deletedAt, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type noteScanner interface {
	Scan(dest ...any) error
}

func scanNote(scanner noteScanner) (domain.Note, error) {
	var note domain.Note
	var tags string
	var pinned int
	if err := scanner.Scan(
		&note.ID,
		&note.Title,
		&note.FilePath,
		&note.Summary,
		&tags,
		&note.Status,
		&pinned,
		&note.CreatedAt,
		&note.UpdatedAt,
		&note.DeletedAt,
	); err != nil {
		return domain.Note{}, err
	}
	if strings.TrimSpace(tags) == "" {
		tags = "[]"
	}
	if err := json.Unmarshal([]byte(tags), &note.Tags); err != nil {
		return domain.Note{}, err
	}
	note.Pinned = pinned != 0
	return note, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
