package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"navigation/internal/domain"
)

const (
	DefaultUsername = "admin"
	DefaultPassword = "admin"

	defaultSiteTitle = "导航站"
	defaultBadge     = "DEV PORTAL / 个人导航站"
	defaultSubtitle  = "聚合了常用网站"
	defaultHeroTitle = "常用站点导航"
	defaultTheme     = "dark"
	sessionTTL       = 24 * time.Hour
)

// AccountStore 定义账号和设置的持久化能力。
type AccountStore interface {
	GetUser() (domain.User, error)
	SaveUser(domain.User) error
	GetSettings() (domain.AppSettings, error)
	SaveSettings(domain.AppSettings) error
}

// AuthService 封装单用户鉴权、会话和显示设置。
type AuthService struct {
	mu       sync.Mutex
	store    AccountStore
	sessions map[string]session
}

type session struct {
	Username  string
	ExpiresAt time.Time
}

// NewAuthService 创建鉴权服务，并确保默认账号存在。
func NewAuthService(store AccountStore) (*AuthService, error) {
	service := &AuthService{store: store, sessions: map[string]session{}}
	if err := service.EnsureDefaultUser(); err != nil {
		return nil, err
	}
	return service, nil
}

// EnsureDefaultUser 在没有账号时初始化 admin/admin。
func (s *AuthService) EnsureDefaultUser() error {
	_, err := s.store.GetUser()
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return s.ResetDefaultUser()
}

// ResetDefaultUser 重置账号密码为 admin/admin。
func (s *AuthService) ResetDefaultUser() error {
	user, err := newUser(DefaultUsername, DefaultPassword)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = map[string]session{}
	return s.store.SaveUser(user)
}

// Login 校验账号密码并创建会话。
func (s *AuthService) Login(username, password string) (string, domain.User, error) {
	username = strings.TrimSpace(username)
	user, err := s.store.GetUser()
	if err != nil {
		return "", domain.User{}, StoreError{Op: StoreOpRead, Err: err}
	}
	if username != user.Username || !verifyPassword(password, user.PasswordSalt, user.PasswordHash) {
		return "", domain.User{}, ValidationError{Message: "账号或密码不正确"}
	}

	token, err := randomToken(32)
	if err != nil {
		return "", domain.User{}, err
	}
	s.mu.Lock()
	s.sessions[token] = session{Username: user.Username, ExpiresAt: time.Now().Add(sessionTTL)}
	s.mu.Unlock()
	return token, user, nil
}

// Logout 删除指定会话。
func (s *AuthService) Logout(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// UserBySession 返回当前会话用户。
func (s *AuthService) UserBySession(token string) (domain.User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.sessions[token]
	if !ok || time.Now().After(current.ExpiresAt) {
		delete(s.sessions, token)
		return domain.User{}, false
	}
	user, err := s.store.GetUser()
	if err != nil || user.Username != current.Username {
		delete(s.sessions, token)
		return domain.User{}, false
	}
	return user, true
}

// UpdateAccount 修改账号和密码，密码为空时保留旧密码。
func (s *AuthService) UpdateAccount(username, currentPassword, newPassword string) (domain.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return domain.User{}, ValidationError{Message: "账号不能为空"}
	}
	if currentPassword == "" {
		return domain.User{}, ValidationError{Message: "当前密码不能为空"}
	}

	user, err := s.store.GetUser()
	if err != nil {
		return domain.User{}, StoreError{Op: StoreOpRead, Err: err}
	}
	if !verifyPassword(currentPassword, user.PasswordSalt, user.PasswordHash) {
		return domain.User{}, ValidationError{Message: "当前密码不正确"}
	}

	user.Username = username
	if strings.TrimSpace(newPassword) != "" {
		updated, err := newUser(username, newPassword)
		if err != nil {
			return domain.User{}, err
		}
		user = updated
	}
	if err := s.store.SaveUser(user); err != nil {
		return domain.User{}, StoreError{Op: StoreOpSave, Err: err}
	}

	s.mu.Lock()
	for token, current := range s.sessions {
		current.Username = user.Username
		s.sessions[token] = current
	}
	s.mu.Unlock()
	return user, nil
}

// Settings 返回显示设置，自动补齐默认值。
func (s *AuthService) Settings() (domain.AppSettings, error) {
	settings, err := s.store.GetSettings()
	if err != nil {
		return domain.AppSettings{}, StoreError{Op: StoreOpRead, Err: err}
	}
	return normalizeSettings(settings), nil
}

// UpdateSettings 保存显示设置。
func (s *AuthService) UpdateSettings(settings domain.AppSettings) (domain.AppSettings, error) {
	settings = normalizeSettings(settings)
	if err := s.store.SaveSettings(settings); err != nil {
		return domain.AppSettings{}, StoreError{Op: StoreOpSave, Err: err}
	}
	return settings, nil
}

func newUser(username, password string) (domain.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || strings.TrimSpace(password) == "" {
		return domain.User{}, ValidationError{Message: "账号和密码不能为空"}
	}
	salt, err := randomToken(16)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{Username: username, PasswordSalt: salt, PasswordHash: hashPassword(password, salt)}, nil
}

func normalizeSettings(settings domain.AppSettings) domain.AppSettings {
	settings.SiteTitle = strings.TrimSpace(settings.SiteTitle)
	settings.Badge = strings.TrimSpace(settings.Badge)
	settings.Subtitle = strings.TrimSpace(settings.Subtitle)
	settings.HeroTitle = strings.TrimSpace(settings.HeroTitle)
	settings.Theme = strings.TrimSpace(settings.Theme)
	if settings.SiteTitle == "" {
		settings.SiteTitle = defaultSiteTitle
	}
	if settings.Badge == "" {
		settings.Badge = defaultBadge
	}
	if settings.Subtitle == "" {
		settings.Subtitle = defaultSubtitle
	}
	if settings.HeroTitle == "" {
		settings.HeroTitle = defaultHeroTitle
	}
	if settings.Theme != "morning" && settings.Theme != "forest" && settings.Theme != "plum" {
		settings.Theme = defaultTheme
	}
	return settings
}

func hashPassword(password, salt string) string {
	key := []byte(password)
	data := []byte(salt)
	for i := 0; i < 120000; i++ {
		h := hmac.New(sha256.New, key)
		h.Write(data)
		data = h.Sum(nil)
	}
	return hex.EncodeToString(data)
}

func verifyPassword(password, salt, expected string) bool {
	actual := hashPassword(password, salt)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
