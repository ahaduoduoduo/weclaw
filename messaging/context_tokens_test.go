package messaging

import "testing"

func TestContextTokenStoreScopesTokensByAccountAndUser(t *testing.T) {
	store := NewContextTokenStore()
	store.Store("account-a", "user-1", "token-a")
	store.Store("account-b", "user-1", "token-b")
	store.Store("account-a", "user-2", "token-c")

	tests := []struct {
		accountID string
		userID    string
		want      string
	}{
		{"account-a", "user-1", "token-a"},
		{"account-b", "user-1", "token-b"},
		{"account-a", "user-2", "token-c"},
	}
	for _, test := range tests {
		got, ok := store.Load(test.accountID, test.userID)
		if !ok || got != test.want {
			t.Fatalf(
				"Load(%q, %q) = %q, %v; want %q, true",
				test.accountID,
				test.userID,
				got,
				ok,
				test.want,
			)
		}
	}
}

func TestContextTokenStoreDoesNotReplaceTokenWithEmptyValue(t *testing.T) {
	store := NewContextTokenStore()
	store.Store("account", "user", "current")
	store.Store("account", "user", "")

	got, ok := store.Load("account", "user")
	if !ok || got != "current" {
		t.Fatalf("Load() = %q, %v; want current, true", got, ok)
	}
}
