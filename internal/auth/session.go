package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"keenetic-go-vpn/internal/config"

	"github.com/gin-gonic/gin"
)

const SessionCookieName = "kgovpn_session"

type Session struct {
	ID        string
	Username  string
	ExpiresAt time.Time
}

type Manager struct {
	user string
	pass string
	ttl  time.Duration

	mu       sync.RWMutex
	sessions map[string]*Session
}

// Supports Go durations ("1h", "30m") and "Nd" = N days.
// Any invalid/empty value is a hard error.
func parseTTL(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty TTL")
	}

	if strings.HasSuffix(s, "d") {
		nStr := strings.TrimSuffix(s, "d")
		n, err := strconv.Atoi(nStr)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("invalid day TTL: %q", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid TTL: %q: %w", s, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("non-positive TTL: %q", s)
	}
	return d, nil
}

func NewManager(cfg config.Config) *Manager {
	ttl, err := parseTTL(cfg.WebTTL)
	if err != nil {
		log.Fatalf("invalid WEB_SESSION_TTL %q: %v", cfg.WebTTL, err)
	}

	return &Manager{
		user:     cfg.WebUser,
		pass:     cfg.WebPass,
		ttl:      ttl,
		sessions: make(map[string]*Session),
	}
}

func (m *Manager) generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *Manager) createSession(username string) (*Session, error) {
	id, err := m.generateSessionID()
	if err != nil {
		return nil, err
	}
	sess := &Session{
		ID:        id,
		Username:  username,
		ExpiresAt: time.Now().Add(m.ttl),
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = sess
	return sess, nil
}

func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	sess, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(sess.ExpiresAt) {
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
		return nil, false
	}
	return sess, true
}

func (m *Manager) RefreshSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sess, ok := m.sessions[id]; ok {
		sess.ExpiresAt = time.Now().Add(m.ttl)
	}
}

func (m *Manager) DeleteSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

func (m *Manager) Login(username, password string) (*Session, error) {
	if username != m.user || password != m.pass {
		return nil, errors.New("invalid credentials")
	}
	return m.createSession(username)
}

func (m *Manager) Logout(sessionID string) {
	m.DeleteSession(sessionID)
}

// Middleware to protect /api/* endpoints.
func (m *Manager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(SessionCookieName)
		if err != nil || cookie == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		sess, ok := m.GetSession(cookie)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		m.RefreshSession(cookie)
		c.Set("sessionUser", sess.Username)
		c.Next()
	}
}

type Handler struct {
	mgr *Manager
}

func NewHandler(mgr *Manager) *Handler {
	return &Handler{mgr: mgr}
}

// POST /api/login
func (h *Handler) Login(c *gin.Context) {
	var req struct {
		User string `json:"user"`
		Pass string `json:"pass"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	sess, err := h.mgr.Login(req.User, req.Pass)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	httpOnly := true
	secure := false // set true if serving over HTTPS
	sameSite := http.SameSiteLaxMode

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: sameSite,
		Expires:  sess.ExpiresAt,
	})

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// POST /api/logout
func (h *Handler) Logout(c *gin.Context) {
	cookie, _ := c.Cookie(SessionCookieName)
	if cookie != "" {
		h.mgr.Logout(cookie)
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     SessionCookieName,
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}