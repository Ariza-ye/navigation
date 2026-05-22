package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	// dataFileName 是站点数据库在数据目录中的固定文件名。
	dataFileName       = "sites.db"
	legacyJSONFileName = "sites.json"
	defaultDataDir     = "data"
	defaultPort        = 8080
	defaultGlow        = "rgba(96,165,250,.45)"
)

// Site 表示导航站中的一个站点条目，并直接映射到前端接口的 JSON 字段。
type Site struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Category    string `json:"category"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Glow        string `json:"glow"`
	Sort        int    `json:"sort"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// Stats 是首页概览统计接口返回的数据结构。
type Stats struct {
	SiteCount     int    `json:"siteCount"`
	CategoryCount int    `json:"categoryCount"`
	Coverage      string `json:"coverage"`
}

// CategoryStat 表示单个分类下的站点数量。
type CategoryStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// app 保存应用运行时依赖。mu 用来保护需要读改写多条记录的操作。
type app struct {
	mu       sync.Mutex
	db       *sql.DB
	dataPath string
	jsonPath string
}

func main() {
	// 通过命令行参数允许调用者覆盖端口和数据目录，默认使用 ./data/sites.db。
	port := flag.Int("port", defaultPort, "HTTP server port")
	dataDir := flag.String("data", defaultDataDir, "directory for SQLite data file")
	flag.Parse()
	if *port < 1 || *port > 65535 {
		log.Fatalf("端口必须在 1 到 65535 之间: %d", *port)
	}
	cleanDataDir := filepath.Clean(strings.TrimSpace(*dataDir))
	if cleanDataDir == "." && strings.TrimSpace(*dataDir) == "" {
		log.Fatal("数据目录不能为空")
	}

	a := &app{
		dataPath: filepath.Join(cleanDataDir, dataFileName),
		jsonPath: filepath.Join(cleanDataDir, legacyJSONFileName),
	}
	if err := a.ensureDatabase(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer a.db.Close()

	// 注册 API 路由和静态首页路由。
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sites", a.handleSites)
	mux.HandleFunc("/api/sites/", a.handleSiteByID)
	mux.HandleFunc("/api/categories", a.handleCategories)
	mux.HandleFunc("/api/categories/", a.handleCategoryByName)
	mux.HandleFunc("/api/category-stats", a.handleCategoryStats)
	mux.HandleFunc("/api/stats", a.handleStats)
	mux.HandleFunc("/", serveIndex)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("导航站已启动: http://localhost%s", addr)
	log.Printf("SQLite 数据库: %s", a.dataPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// serveIndex 只提供首页文件；其他未匹配路径返回 404，避免误把 API 路径当成静态文件。
func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "index.html")
}

// handleSites 处理站点集合接口：GET 查询列表，POST 创建站点。
func (a *app) handleSites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.listSites(w, r)
	case http.MethodPost:
		a.createSite(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

// handleSiteByID 处理单个站点接口，站点 ID 从 /api/sites/{id} 中取得。
func (a *app) handleSiteByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	id = strings.TrimSpace(id)
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少站点 ID")
		return
	}

	switch r.Method {
	case http.MethodPut:
		a.updateSite(w, r, id)
	case http.MethodDelete:
		a.deleteSite(w, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

// listSites 支持按分类和关键字过滤站点，并在返回前统一排序。
func (a *app) listSites(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))

	sites, err := a.loadSites()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}

	filtered := make([]Site, 0, len(sites))
	for _, site := range sites {
		// “全部”是前端使用的虚拟分类，不参与后端实际分类匹配。
		if category != "" && category != "全部" && site.Category != category {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(site.Name + " " + site.Description + " " + site.Category)
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		filtered = append(filtered, site)
	}

	sortSites(filtered)
	writeJSON(w, http.StatusOK, filtered)
}

// createSite 创建新站点。写操作需要加锁，确保默认排序值和插入动作连续完成。
func (a *app) createSite(w http.ResponseWriter, r *http.Request) {
	var input Site
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求数据格式不正确")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	sites, err := a.loadSitesLocked()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}

	now := time.Now().Format(time.RFC3339)
	// ID 和时间戳由后端生成，避免客户端伪造或遗漏关键字段。
	input.ID = newID()
	input.CreatedAt = now
	input.UpdatedAt = now
	normalizeSite(&input, nextSort(sites))

	if err := validateSite(input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sites = append(sites, input)
	sortSites(sites)
	if err := a.saveSitesLocked(sites); err != nil {
		writeError(w, http.StatusInternalServerError, "保存站点数据失败")
		return
	}

	writeJSON(w, http.StatusCreated, input)
}

// updateSite 使用请求体中的内容替换指定站点，但保留原 ID 和创建时间。
func (a *app) updateSite(w http.ResponseWriter, r *http.Request, id string) {
	var input Site
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求数据格式不正确")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	sites, err := a.loadSitesLocked()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}

	for i, site := range sites {
		if site.ID != id {
			continue
		}
		input.ID = site.ID
		input.CreatedAt = site.CreatedAt
		input.UpdatedAt = time.Now().Format(time.RFC3339)
		normalizeSite(&input, site.Sort)
		if err := validateSite(input); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		sites[i] = input
		sortSites(sites)
		if err := a.saveSitesLocked(sites); err != nil {
			writeError(w, http.StatusInternalServerError, "保存站点数据失败")
			return
		}
		writeJSON(w, http.StatusOK, input)
		return
	}

	writeError(w, http.StatusNotFound, "没有找到这个站点")
}

// deleteSite 从站点列表中删除指定 ID 的条目。
func (a *app) deleteSite(w http.ResponseWriter, id string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sites, err := a.loadSitesLocked()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}

	for i, site := range sites {
		if site.ID != id {
			continue
		}
		sites = append(sites[:i], sites[i+1:]...)
		if err := a.saveSitesLocked(sites); err != nil {
			writeError(w, http.StatusInternalServerError, "保存站点数据失败")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeError(w, http.StatusNotFound, "没有找到这个站点")
}

// handleCategories 返回当前存在的分类列表，并在开头追加前端需要的“全部”选项。
func (a *app) handleCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	sites, err := a.loadSites()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}

	seen := map[string]bool{}
	categories := []string{"全部"}
	for _, site := range sites {
		if site.Category == "" || seen[site.Category] {
			continue
		}
		seen[site.Category] = true
		categories = append(categories, site.Category)
	}
	writeJSON(w, http.StatusOK, categories)
}

// handleCategoryByName 处理分类删除请求，分类名称来自 URL 路径。
func (a *app) handleCategoryByName(w http.ResponseWriter, r *http.Request) {
	name, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/api/categories/"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "分类名称不正确")
		return
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "全部" {
		writeError(w, http.StatusBadRequest, "不能删除这个分类")
		return
	}

	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	a.deleteCategory(w, name)
}

// deleteCategory 不会删除站点本身，只是把属于该分类的站点改为未分类。
func (a *app) deleteCategory(w http.ResponseWriter, name string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sites, err := a.loadSitesLocked()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}

	updated := 0
	now := time.Now().Format(time.RFC3339)
	for i := range sites {
		if sites[i].Category == name {
			sites[i].Category = ""
			sites[i].UpdatedAt = now
			updated++
		}
	}

	if updated == 0 {
		writeError(w, http.StatusNotFound, "没有找到这个分类")
		return
	}

	if err := a.saveSitesLocked(sites); err != nil {
		writeError(w, http.StatusInternalServerError, "保存站点数据失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"uncategorizedSites": updated})
}

// handleCategoryStats 返回每个分类的站点数量，用于前端分类统计展示。
func (a *app) handleCategoryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	sites, err := a.loadSites()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}

	counts := map[string]int{}
	for _, site := range sites {
		if site.Category != "" {
			counts[site.Category]++
		}
	}

	categories := make([]CategoryStat, 0, len(counts))
	for name, count := range counts {
		categories = append(categories, CategoryStat{Name: name, Count: count})
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})

	writeJSON(w, http.StatusOK, categories)
}

// handleStats 返回全站聚合统计数据。
func (a *app) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	sites, err := a.loadSites()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}

	categories := map[string]bool{}
	for _, site := range sites {
		if site.Category != "" {
			categories[site.Category] = true
		}
	}

	writeJSON(w, http.StatusOK, Stats{
		SiteCount:     len(sites),
		CategoryCount: len(categories),
		Coverage:      "99%",
	})
}

// ensureDatabase 确保 SQLite 数据库和表结构存在；空库首次启动时会从旧 JSON 文件导入数据。
func (a *app) ensureDatabase() error {
	if err := os.MkdirAll(filepath.Dir(a.dataPath), 0755); err != nil {
		return err
	}

	db, err := sql.Open("sqlite3", a.dataPath)
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
	`); err != nil {
		db.Close()
		return err
	}

	a.db = db
	return a.importLegacyJSONIfNeeded()
}

// importLegacyJSONIfNeeded 只在数据库为空时导入旧的 sites.json，便于平滑升级。
func (a *app) importLegacyJSONIfNeeded() error {
	var count int
	if err := a.db.QueryRow("SELECT COUNT(*) FROM sites").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if _, err := os.Stat(a.jsonPath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	data, err := os.ReadFile(a.jsonPath)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}

	var sites []Site
	if err := json.Unmarshal(data, &sites); err != nil {
		return err
	}
	sortSites(sites)
	return a.saveSitesLocked(sites)
}

// loadSites 是带锁的读取入口，适合请求处理函数直接调用。
func (a *app) loadSites() ([]Site, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.loadSitesLocked()
}

// loadSitesLocked 从 SQLite 读取站点数据。调用者必须已经持有 a.mu。
func (a *app) loadSitesLocked() ([]Site, error) {
	rows, err := a.db.Query(`
		SELECT id, name, url, category, icon, description, glow, sort, created_at, updated_at
		FROM sites
		ORDER BY sort ASC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sites := []Site{}
	for rows.Next() {
		var site Site
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

// saveSitesLocked 用事务整体替换站点数据，保持读改写操作的原子性。
func (a *app) saveSitesLocked(sites []Site) error {
	tx, err := a.db.Begin()
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

// normalizeSite 清理用户输入，并为可选字段补上默认值。
func normalizeSite(site *Site, fallbackSort int) {
	site.Name = strings.TrimSpace(site.Name)
	site.URL = strings.TrimSpace(site.URL)
	site.Category = strings.TrimSpace(site.Category)
	site.Icon = strings.TrimSpace(site.Icon)
	site.Description = strings.TrimSpace(site.Description)
	site.Glow = strings.TrimSpace(site.Glow)

	if site.Icon == "" {
		site.Icon = "🔗"
	}
	if site.Glow == "" {
		site.Glow = defaultGlow
	}
	if site.Sort <= 0 {
		site.Sort = fallbackSort
	}
}

// validateSite 校验站点必填字段和 URL 格式，返回的错误会直接展示给前端。
func validateSite(site Site) error {
	if site.Name == "" {
		return errors.New("站点名称不能为空")
	}
	if site.URL == "" {
		return errors.New("站点地址不能为空")
	}
	parsed, err := url.ParseRequestURI(site.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("站点地址格式不正确")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("站点地址必须以 http:// 或 https:// 开头")
	}
	if site.Category == "" {
		return errors.New("站点分类不能为空")
	}
	return nil
}

// nextSort 生成默认排序值，让新站点排在现有站点之后。
func nextSort(sites []Site) int {
	maxSort := 0
	for _, site := range sites {
		if site.Sort > maxSort {
			maxSort = site.Sort
		}
	}
	return maxSort + 1
}

// sortSites 按 Sort 升序排列；排序值相同时按名称排列，保证结果稳定可预期。
func sortSites(sites []Site) {
	sort.SliceStable(sites, func(i, j int) bool {
		if sites[i].Sort == sites[j].Sort {
			return sites[i].Name < sites[j].Name
		}
		return sites[i].Sort < sites[j].Sort
	})
}

// newID 优先使用随机字节生成短 ID；随机源失败时退化为时间戳 ID。
func newID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "site_" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("site_%d", time.Now().UnixNano())
}

// writeJSON 统一写入 JSON 响应和 Content-Type。
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("写入响应失败: %v", err)
	}
}

// writeError 统一错误响应格式，方便前端按 error 字段读取错误信息。
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
