package management

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fastclaw-ai/weclaw/config"
)

type User struct {
	AccountID    string    `json:"account_id"`
	UserID       string    `json:"user_id"`
	Status       string    `json:"status"`
	DefaultAgent string    `json:"default_agent,omitempty"`
	FirstSeenAt  time.Time `json:"first_seen_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

type accessData struct {
	Users []User `json:"users"`
}

// AccessStore records discovered contacts and applies agent allowlists uniformly.
type AccessStore struct {
	mu      sync.RWMutex
	path    string
	users   map[string]User
	configs *config.RuntimeStore
}

func NewAccessStore(configs *config.RuntimeStore) (*AccessStore, error) {
	path, err := statePath("access.json")
	if err != nil {
		return nil, err
	}
	store := &AccessStore{
		path:    path,
		users:   make(map[string]User),
		configs: configs,
	}
	if data, err := os.ReadFile(path); err == nil {
		var saved accessData
		if json.Unmarshal(data, &saved) == nil {
			for _, user := range saved.Users {
				store.users[userKey(user.AccountID, user.UserID)] = user
			}
		}
	}
	return store, nil
}

func (s *AccessStore) Observe(accountID, userID string) {
	if accountID == "" || userID == "" {
		return
	}
	now := time.Now().UTC()
	s.mu.Lock()
	key := userKey(accountID, userID)
	user, ok := s.users[key]
	if !ok {
		user = User{
			AccountID:   accountID,
			UserID:      userID,
			Status:      initialStatus(s.configs.Agents()),
			FirstSeenAt: now,
		}
	}
	user.LastSeenAt = now
	s.users[key] = user
	_ = s.saveLocked()
	s.mu.Unlock()
}

func (s *AccessStore) Allowed(accountID, userID, agentName string) bool {
	s.mu.RLock()
	user, known := s.users[userKey(accountID, userID)]
	s.mu.RUnlock()
	if known && user.Status == "blocked" {
		return false
	}
	agent, ok := s.configs.Agent(agentName)
	if !ok {
		return false
	}
	for _, allowed := range agent.AllowedUsers {
		if allowed == "*" || allowed == userID ||
			allowed == accountID+":"+userID {
			return true
		}
	}
	return false
}

func (s *AccessStore) DefaultAgent(accountID, userID, global string) string {
	s.mu.RLock()
	user, ok := s.users[userKey(accountID, userID)]
	s.mu.RUnlock()
	if ok && user.DefaultAgent != "" &&
		s.Allowed(accountID, userID, user.DefaultAgent) {
		return user.DefaultAgent
	}
	return global
}

func (s *AccessStore) SetDefaultAgent(accountID, userID, agentName string) error {
	if !s.Allowed(accountID, userID, agentName) {
		return fmt.Errorf("agent %q is not permitted for this user", agentName)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userKey(accountID, userID)
	user, ok := s.users[key]
	if !ok {
		return fmt.Errorf("user not found")
	}
	user.DefaultAgent = agentName
	s.users[key] = user
	return s.saveLocked()
}

func (s *AccessStore) Users() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]User, 0, len(s.users))
	for _, user := range s.users {
		result = append(result, user)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeenAt.After(result[j].LastSeenAt)
	})
	return result
}

func (s *AccessStore) UpdateUser(accountID, userID, status, defaultAgent string) error {
	if status != "pending" && status != "active" && status != "blocked" {
		return fmt.Errorf("invalid user status")
	}
	if defaultAgent != "" && !s.Allowed(accountID, userID, defaultAgent) {
		return fmt.Errorf("default agent is not permitted")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userKey(accountID, userID)
	user, ok := s.users[key]
	if !ok {
		return fmt.Errorf("user not found")
	}
	user.Status = status
	user.DefaultAgent = defaultAgent
	s.users[key] = user
	return s.saveLocked()
}

func (s *AccessStore) SetPermission(accountID, userID, agentName string, allowed bool) error {
	agent, ok := s.configs.Agent(agentName)
	if !ok {
		return fmt.Errorf("agent not found")
	}
	target := accountID + ":" + userID
	next := make([]string, 0, len(agent.AllowedUsers)+1)
	for _, value := range agent.AllowedUsers {
		if value != target && value != userID {
			next = append(next, value)
		}
	}
	if allowed {
		next = append(next, target)
	}
	agent.AllowedUsers = next
	return s.configs.SaveAgent(agentName, agent)
}

func (s *AccessStore) SetPublic(agentName string, public bool) error {
	agent, ok := s.configs.Agent(agentName)
	if !ok {
		return fmt.Errorf("agent not found")
	}
	next := make([]string, 0, len(agent.AllowedUsers)+1)
	for _, value := range agent.AllowedUsers {
		if value != "*" {
			next = append(next, value)
		}
	}
	if public {
		next = append([]string{"*"}, next...)
	}
	agent.AllowedUsers = next
	return s.configs.SaveAgent(agentName, agent)
}

func (s *AccessStore) saveLocked() error {
	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	data, err := json.MarshalIndent(accessData{Users: users}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func statePath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".weclaw", name), nil
}

func userKey(accountID, userID string) string {
	return accountID + "\x00" + userID
}

func initialStatus(agents map[string]config.AgentConfig) string {
	for _, agent := range agents {
		for _, user := range agent.AllowedUsers {
			if user == "*" {
				return "active"
			}
		}
	}
	return "pending"
}
