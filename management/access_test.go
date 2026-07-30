package management

import (
	"testing"

	"github.com/fastclaw-ai/weclaw/config"
)

func TestAccessStoreAppliesPublicAndPerAccountPermissions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configs := config.NewRuntimeStore(&config.Config{
		DefaultAgent: "public",
		Agents: map[string]config.AgentConfig{
			"public":  {Type: "native", Endpoint: "http://public", AllowedUsers: []string{"*"}},
			"private": {Type: "http", Endpoint: "http://private"},
		},
	})
	store, err := NewAccessStore(configs)
	if err != nil {
		t.Fatal(err)
	}
	store.Observe("bot-a", "user-1")

	if !store.Allowed("bot-a", "user-1", "public") {
		t.Fatal("public agent should be available")
	}
	if store.Allowed("bot-a", "user-1", "private") {
		t.Fatal("private agent should default to denied")
	}
	if err := store.SetPermission("bot-a", "user-1", "private", true); err != nil {
		t.Fatal(err)
	}
	if !store.Allowed("bot-a", "user-1", "private") {
		t.Fatal("explicit account/user permission was not applied")
	}
	if store.Allowed("bot-b", "user-1", "private") {
		t.Fatal("permission must not leak to another WeChat account")
	}
}

func TestAccessStorePersistsPerUserDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configs := config.NewRuntimeStore(&config.Config{
		DefaultAgent: "a",
		Agents: map[string]config.AgentConfig{
			"a": {Type: "native", AllowedUsers: []string{"*"}},
			"b": {Type: "native", AllowedUsers: []string{"*"}},
		},
	})
	store, _ := NewAccessStore(configs)
	store.Observe("bot", "user")
	if err := store.SetDefaultAgent("bot", "user", "b"); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewAccessStore(configs)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.DefaultAgent("bot", "user", "a"); got != "b" {
		t.Fatalf("default agent = %q, want b", got)
	}
}
