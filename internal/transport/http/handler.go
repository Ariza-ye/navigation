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
	static  fs.FS
}

// NewHandler 创建 HTTP 处理器。
func NewHandler(service *service.SiteService, static fs.FS) *Handler {
	return &Handler{service: service, static: static}
}

// Routes 注册页面和 API 路由。
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sites", h.handleSites)
	mux.HandleFunc("/api/sites/", h.handleSiteByID)
	mux.HandleFunc("/api/categories", h.handleCategories)
	mux.HandleFunc("/api/categories/", h.handleCategoryByName)
	mux.HandleFunc("/api/category-stats", h.handleCategoryStats)
	mux.HandleFunc("/api/stats", h.handleStats)
	mux.HandleFunc("/", h.serveIndex)
	return mux
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	http.ServeFileFS(w, r, h.static, "index.html")
}

func (h *Handler) handleSites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listSites(w, r)
	case http.MethodPost:
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
		writeError(w, http.StatusBadRequest, "不能删除这个分类")
		return
	}

	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
		return
	}

	updated, err := h.service.DeleteCategory(name)
	if err != nil {
		h.writeServiceError(w, err, "没有找到这个分类")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"uncategorizedSites": updated})
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
