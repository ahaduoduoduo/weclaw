package management

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const sessionTTL = 24 * time.Hour

type authData struct {
	Salt string `json:"salt"`
	Hash string `json:"hash"`
}

type Auth struct {
	mu       sync.Mutex
	path     string
	password *authData
	sessions map[string]time.Time
}

func NewAuth() (*Auth, error) {
	path, err := statePath("admin.json")
	if err != nil {
		return nil, err
	}
	auth := &Auth{path: path, sessions: make(map[string]time.Time)}
	if raw, err := os.ReadFile(path); err == nil {
		var saved authData
		if json.Unmarshal(raw, &saved) == nil && saved.Hash != "" {
			auth.password = &saved
		}
	}
	return auth, nil
}

func (a *Auth) SetupRequired() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.password == nil
}

func (a *Auth) Setup(password string) (string, error) {
	if len(password) < 10 {
		return "", fmt.Errorf("password must contain at least 10 characters")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.password != nil {
		return "", fmt.Errorf("administrator is already configured")
	}
	salt := randomBytes(16)
	saved := authData{
		Salt: base64.RawURLEncoding.EncodeToString(salt),
		Hash: base64.RawURLEncoding.EncodeToString(derivePassword(password, salt)),
	}
	raw, _ := json.MarshalIndent(saved, "", "  ")
	if err := os.MkdirAll(filepath.Dir(a.path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(a.path, raw, 0o600); err != nil {
		return "", err
	}
	a.password = &saved
	return a.createSessionLocked(), nil
}

func (a *Auth) Login(password string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.password == nil {
		return "", fmt.Errorf("administrator is not configured")
	}
	salt, err := base64.RawURLEncoding.DecodeString(a.password.Salt)
	if err != nil {
		return "", err
	}
	expected, err := base64.RawURLEncoding.DecodeString(a.password.Hash)
	if err != nil {
		return "", err
	}
	actual := derivePassword(password, salt)
	if !hmac.Equal(actual, expected) {
		return "", fmt.Errorf("invalid password")
	}
	return a.createSessionLocked(), nil
}

func (a *Auth) Valid(token string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	expiresAt, ok := a.sessions[token]
	if !ok || time.Now().After(expiresAt) {
		delete(a.sessions, token)
		return false
	}
	return true
}

func (a *Auth) Logout(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, token)
}

func (a *Auth) createSessionLocked() string {
	token := base64.RawURLEncoding.EncodeToString(randomBytes(32))
	a.sessions[token] = time.Now().Add(sessionTTL)
	return token
}

func derivePassword(password string, salt []byte) []byte {
	value := append(append([]byte(nil), salt...), []byte(password)...)
	for i := 0; i < 150_000; i++ {
		sum := sha256.Sum256(value)
		value = sum[:]
	}
	return value
}

func randomBytes(size int) []byte {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		panic(err)
	}
	return value
}
