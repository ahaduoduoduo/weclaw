package messaging

import (
	"strings"
	"sync"
)

// ContextTokenStore keeps the latest iLink context token for each account and
// contact. Tokens remain process-local and are refreshed by incoming messages.
type ContextTokenStore struct {
	values sync.Map
}

func NewContextTokenStore() *ContextTokenStore {
	return &ContextTokenStore{}
}

func (s *ContextTokenStore) Store(accountID, userID, token string) {
	if s == nil ||
		strings.TrimSpace(accountID) == "" ||
		strings.TrimSpace(userID) == "" ||
		strings.TrimSpace(token) == "" {
		return
	}
	s.values.Store(contextTokenKey(accountID, userID), token)
}

func (s *ContextTokenStore) Load(accountID, userID string) (string, bool) {
	if s == nil {
		return "", false
	}
	value, ok := s.values.Load(contextTokenKey(accountID, userID))
	if !ok {
		return "", false
	}
	token, ok := value.(string)
	return token, ok && token != ""
}

func contextTokenKey(accountID, userID string) string {
	return accountID + "\x00" + userID
}
