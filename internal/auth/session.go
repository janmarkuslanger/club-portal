package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Session struct {
	UserID    string
	ExpiresAt time.Time
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]Session
	ttl      time.Duration
}

func NewManager(ttl time.Duration) *Manager {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	return &Manager{
		sessions: make(map[string]Session),
		ttl:      ttl,
	}
}

func (m *Manager) Create(userID string) string {
	token := newToken()
	m.mu.Lock()
	m.sessions[token] = Session{
		UserID:    userID,
		ExpiresAt: time.Now().Add(m.ttl),
	}
	m.mu.Unlock()

	return token
}

func (m *Manager) Get(token string) (string, bool) {
	m.mu.RLock()
	session, ok := m.sessions[token]
	m.mu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Now().After(session.ExpiresAt) {
		m.Delete(token)
		return "", false
	}
	return session.UserID, true
}

func (m *Manager) Delete(token string) {
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
}

func newToken() string {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf[:])
}
