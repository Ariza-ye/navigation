package httptransport

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"navigation/internal/domain"
	"navigation/internal/service"
)

// Handler 负责 HTTP 路由、请求解析和响应编码。
type Handler struct {
	service *service.SiteService
	auth    *service.AuthService
	static  fs.FS
}

// NewHandler 创建 HTTP 处理器。
func NewHandler(service *service.SiteService, auth *service.AuthService, static fs.FS) *Handler {
	return &Handler{service: service, auth: auth, static: static}
}

// Routes 注册页面和 API 路由。
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", h.handleLogin)
	mux.HandleFunc("/api/session", h.handleSession)
	mux.HandleFunc("/api/logout", h.requireAuth(h.handleLogout))
	mux.HandleFunc("/api/account", h.requireAuth(h.handleAccount))
	mux.HandleFunc("/api/settings", h.handleSettings)
	mux.HandleFunc("/api/sites", h.handleSites)
	mux.HandleFunc("/api/sites/", h.requireAuth(h.handleSiteByID))
	mux.HandleFunc("/api/categories", h.handleCategories)
	mux.HandleFunc("/api/categories/", h.requireAuth(h.handleCategoryByName))
	mux.HandleFunc("/api/category-stats", h.handleCategoryStats)
	mux.HandleFunc("/api/stats", h.handleStats)
	mux.Handle("/assets/", http.FileServerFS(h.static))
	mux.HandleFunc("/", h.serveIndex)
	return mux
}

func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.ensureAuth(w, r) {
			return
		}
		next(w, r)
	}
}

func (h *Handler) ensureAuth(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("navigation_session")
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "请先登录")
		return false
	}
	if _, ok := h.auth.UserBySession(cookie.Value); !ok {
		clearSessionCookie(w)
		writeError(w, http.StatusUnauthorized, "登录已失效")
		return false
	}
	return true
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/assets/") {
		http.NotFound(w, r)
		return
	}
	index, err := fs.ReadFile(h.static, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(index)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求数据格式不正确")
		return
	}
	token, user, err := h.auth.Login(input.Username, input.Password)
	if err != nil {
		h.writeServiceError(w, err, "账号不存在")
		return
	}
	setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]string{"username": user.Username})
}

func (h *Handler) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}
	cookie, err := r.Cookie("navigation_session")
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "请先登录")
		return
	}
	user, ok := h.auth.UserBySession(cookie.Value)
	if !ok {
		clearSessionCookie(w)
		writeError(w, http.StatusUnauthorized, "登录已失效")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": user.Username})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}
	if cookie, err := r.Cookie("navigation_session"); err == nil {
		h.auth.Logout(cookie.Value)
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}
	var input struct {
		Username        string `json:"username"`
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求数据格式不正确")
		return
	}
	user, err := h.auth.UpdateAccount(input.Username, input.CurrentPassword, input.NewPassword)
	if err != nil {
		h.writeServiceError(w, err, "账号不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": user.Username})
}

func (h *Handler) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := h.auth.Settings()
		if err != nil {
			h.writeServiceError(w, err, "没有找到设置")
			return
		}
		writeJSON(w, http.StatusOK, settings)
	case http.MethodPut:
		if !h.ensureAuth(w, r) {
			return
		}
		var input domain.AppSettings
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求数据格式不正确")
			return
		}
		settings, err := h.auth.UpdateSettings(input)
		if err != nil {
			h.writeServiceError(w, err, "没有找到设置")
			return
		}
		writeJSON(w, http.StatusOK, settings)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (h *Handler) handleSites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listSites(w, r)
	case http.MethodPost:
		if !h.ensureAuth(w, r) {
			return
		}
		h.createSite(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (h *Handler) handleSiteByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	id = strings.TrimSpace(id)
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少站点 ID")
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.updateSite(w, r, id)
	case http.MethodDelete:
		h.deleteSite(w, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (h *Handler) listSites(w http.ResponseWriter, r *http.Request) {
	sites, err := h.service.ListSites(r.URL.Query().Get("category"), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}
	writeJSON(w, http.StatusOK, sites)
}

func (h *Handler) createSite(w http.ResponseWriter, r *http.Request) {
	var input domain.Site
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求数据格式不正确")
		return
	}

	site, err := h.service.CreateSite(input)
	if err != nil {
		h.writeServiceError(w, err, "没有找到这个站点")
		return
	}
	writeJSON(w, http.StatusCreated, site)
}

func (h *Handler) updateSite(w http.ResponseWriter, r *http.Request, id string) {
	var input domain.Site
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求数据格式不正确")
		return
	}

	site, err := h.service.UpdateSite(id, input)
	if err != nil {
		h.writeServiceError(w, err, "没有找到这个站点")
		return
	}
	writeJSON(w, http.StatusOK, site)
}

func (h *Handler) deleteSite(w http.ResponseWriter, id string) {
	if err := h.service.DeleteSite(id); err != nil {
		h.writeServiceError(w, err, "没有找到这个站点")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	categories, err := h.service.ListCategories()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (h *Handler) handleCategoryByName(w http.ResponseWriter, r *http.Request) {
	name, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/api/categories/"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "分类名称不正确")
		return
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "全部" {
		writeError(w, http.StatusBadRequest, "分类名称不正确")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		updated, err := h.service.DeleteCategory(name)
		if err != nil {
			h.writeServiceError(w, err, "没有找到这个分类")
			return
		}
		writeJSON(w, http.StatusOK, map[string]int{"uncategorizedSites": updated})
	case http.MethodPut:
		var input struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求数据格式不正确")
			return
		}
		updated, err := h.service.RenameCategory(name, input.Name)
		if err != nil {
			h.writeServiceError(w, err, "没有找到这个分类")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"name": strings.TrimSpace(input.Name), "renamedSites": updated})
	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func (h *Handler) handleCategoryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	categories, err := h.service.CategoryStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	stats, err := h.service.Stats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取站点数据失败")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error, notFoundMessage string) {
	var validationErr service.ValidationError
	if errors.As(err, &validationErr) {
		writeError(w, http.StatusBadRequest, validationErr.Error())
		return
	}
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, notFoundMessage)
		return
	}

	var storeErr service.StoreError
	if errors.As(err, &storeErr) {
		if storeErr.Op == service.StoreOpRead {
			writeError(w, http.StatusInternalServerError, "读取站点数据失败")
			return
		}
		writeError(w, http.StatusInternalServerError, "保存站点数据失败")
		return
	}
	writeError(w, http.StatusInternalServerError, "保存站点数据失败")
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "navigation_session",
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "navigation_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
