package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"navigation/internal/domain"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteSiteStore 使用 SQLite 持久化导航站点数据。
type SQLiteSiteStore struct {
	db       *sql.DB
	dataPath string
	jsonPath string
}

// NewSQLiteSiteStore 打开或初始化 SQLite 数据库，必要时从旧 JSON 文件导入数据。
func NewSQLiteSiteStore(dataPath, jsonPath string) (*SQLiteSiteStore, error) {
	store := &SQLiteSiteStore{dataPath: dataPath, jsonPath: jsonPath}
	if err := store.ensureDatabase(); err != nil {
		return nil, err
	}
	return store, nil
}

// Close 关闭底层数据库连接。
func (s *SQLiteSiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteSiteStore) ensureDatabase() error {
	if err := os.MkdirAll(filepath.Dir(s.dataPath), 0755); err != nil {
		return err
	}

	db, err := sql.Open("sqlite3", s.dataPath)
	if err != nil {
		return err
	}

	if _, err := db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = ON;
		CREATE TABLE IF NOT EXISTS sites (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT '',
			icon TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			glow TEXT NOT NULL DEFAULT '',
			sort INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_sites_sort_name ON sites(sort, name);
		CREATE INDEX IF NOT EXISTS idx_sites_category ON sites(category);
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			username TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			password_salt TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`); err != nil {
		db.Close()
		return err
	}

	s.db = db
	return s.importLegacyJSONIfNeeded()
}

// GetUser 读取唯一管理员账号。
func (s *SQLiteSiteStore) GetUser() (domain.User, error) {
	var user domain.User
	err := s.db.QueryRow("SELECT username, password_hash, password_salt FROM users WHERE id = 1").Scan(&user.Username, &user.PasswordHash, &user.PasswordSalt)
	return user, err
}

// SaveUser 覆盖唯一管理员账号。
func (s *SQLiteSiteStore) SaveUser(user domain.User) error {
	_, err := s.db.Exec(`
		INSERT INTO users (id, username, password_hash, password_salt, updated_at)
		VALUES (1, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			username = excluded.username,
			password_hash = excluded.password_hash,
			password_salt = excluded.password_salt,
			updated_at = excluded.updated_at
	`, user.Username, user.PasswordHash, user.PasswordSalt)
	return err
}

// GetSettings 读取站点显示设置。
func (s *SQLiteSiteStore) GetSettings() (domain.AppSettings, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return domain.AppSettings{}, err
	}
	defer rows.Close()

	settings := domain.AppSettings{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return domain.AppSettings{}, err
		}
		switch key {
		case "siteTitle":
			settings.SiteTitle = value
		case "badge":
			settings.Badge = value
		case "subtitle":
			settings.Subtitle = value
		case "heroTitle":
			settings.HeroTitle = value
		case "theme":
			settings.Theme = value
		}
	}
	return settings, rows.Err()
}

// SaveSettings 保存站点显示设置。
func (s *SQLiteSiteStore) SaveSettings(settings domain.AppSettings) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value")
	if err != nil {
		return err
	}
	defer stmt.Close()

	values := map[string]string{
		"siteTitle": settings.SiteTitle,
		"badge":     settings.Badge,
		"subtitle":  settings.Subtitle,
		"heroTitle": settings.HeroTitle,
		"theme":     settings.Theme,
	}
	for key, value := range values {
		if _, err := stmt.Exec(key, value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteSiteStore) importLegacyJSONIfNeeded() error {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM sites").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if _, err := os.Stat(s.jsonPath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	data, err := os.ReadFile(s.jsonPath)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}

	var sites []domain.Site
	if err := json.Unmarshal(data, &sites); err != nil {
		return err
	}
	sortSites(sites)
	return s.SaveSites(sites)
}

// ListSites 读取全部站点，并按排序值和名称升序返回。
func (s *SQLiteSiteStore) ListSites() ([]domain.Site, error) {
	rows, err := s.db.Query(`
		SELECT id, name, url, category, icon, description, glow, sort, created_at, updated_at
		FROM sites
		ORDER BY sort ASC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sites := []domain.Site{}
	for rows.Next() {
		var site domain.Site
		if err := rows.Scan(
			&site.ID,
			&site.Name,
			&site.URL,
			&site.Category,
			&site.Icon,
			&site.Description,
			&site.Glow,
			&site.Sort,
			&site.CreatedAt,
			&site.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	return sites, rows.Err()
}

// SaveSites 在一个事务里用传入列表替换当前全部站点。
func (s *SQLiteSiteStore) SaveSites(sites []domain.Site) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM sites"); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO sites (
			id, name, url, category, icon, description, glow, sort, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, site := range sites {
		if _, err := stmt.Exec(
			site.ID,
			site.Name,
			site.URL,
			site.Category,
			site.Icon,
			site.Description,
			site.Glow,
			site.Sort,
			site.CreatedAt,
			site.UpdatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func sortSites(sites []domain.Site) {
	for i := 1; i < len(sites); i++ {
		for j := i; j > 0; j-- {
			if sites[j-1].Sort < sites[j].Sort || sites[j-1].Sort == sites[j].Sort && sites[j-1].Name <= sites[j].Name {
				break
			}
			sites[j-1], sites[j] = sites[j], sites[j-1]
		}
	}
}
