package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAgentConfigUnmarshalEnv(t *testing.T) {
	var cfg Config
	data := []byte(`{
		"agents": {
			"claude": {
				"type": "cli",
				"command": "claude",
				"env": {
					"ANTHROPIC_API_KEY": "test-key",
					"EMPTY": ""
				}
			}
		}
	}`)

	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	ag, ok := cfg.Agents["claude"]
	if !ok {
		t.Fatalf("expected claude agent config")
	}
	if got := ag.Env["ANTHROPIC_API_KEY"]; got != "test-key" {
		t.Fatalf("ANTHROPIC_API_KEY = %q, want %q", got, "test-key")
	}
	if got, ok := ag.Env["EMPTY"]; !ok || got != "" {
		t.Fatalf("EMPTY = %q, present=%v; want empty string present", got, ok)
	}
}

func TestAgentConfigMarshalEnvRoundTrip(t *testing.T) {
	cfg := Config{
		Agents: map[string]AgentConfig{
			"claude": {
				Type:    "cli",
				Command: "claude",
				Env: map[string]string{
					"ANTHROPIC_API_KEY": "test-key",
					"EMPTY":             "",
				},
			},
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}

	got := decoded.Agents["claude"].Env
	if got["ANTHROPIC_API_KEY"] != "test-key" || got["EMPTY"] != "" {
		t.Fatalf("round-trip env = %#v", got)
	}
}

func TestAgentConfigWithoutEnvStillLoads(t *testing.T) {
	var cfg Config
	data := []byte(`{
		"agents": {
			"claude": {
				"type": "cli",
				"command": "claude"
			}
		}
	}`)

	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config without env: %v", err)
	}

	if cfg.Agents["claude"].Env != nil {
		t.Fatalf("Env = %#v, want nil", cfg.Agents["claude"].Env)
	}
}

func TestNativeAgentConfiguration(t *testing.T) {
	var cfg Config
	data := []byte(`{
		"agents": {
			"movies": {
				"type": "native",
				"endpoint": "http://autofilm-core:3000/v1/conversation/events",
				"api_key": "inbound-secret",
				"outbound_token": "outbound-secret",
				"allowed_users": ["user-1"],
				"timeout_seconds": 180
			}
		}
	}`)

	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal native config: %v", err)
	}
	native := cfg.Agents["movies"]
	if native.Type != "native" ||
		native.OutboundToken != "outbound-secret" ||
		native.TimeoutSeconds != 180 ||
		len(native.AllowedUsers) != 1 {
		t.Fatalf("native config = %#v", native)
	}
}

func TestDefaultConfigInitializesAgentsMap(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Agents == nil {
		t.Fatal("DefaultConfig() Agents = nil, want initialized map")
	}
}

func TestSaveConfigAllowsReadOnlySharingWithServiceGroup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := DefaultConfig()
	cfg.DefaultAgent = "native"
	cfg.Agents["native"] = AgentConfig{Type: "native"}
	configDir := filepath.Join(home, ".weclaw")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write existing config: %v", err)
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("config mode = %o, want 640", got)
	}
}

func TestLoadEnvOverridesTopLevelOnly(t *testing.T) {
	t.Setenv("WECLAW_DEFAULT_AGENT", "codex")
	t.Setenv("WECLAW_API_ADDR", "127.0.0.1:18011")

	cfg := DefaultConfig()
	cfg.Agents["claude"] = AgentConfig{
		Type: "cli",
		Env: map[string]string{
			"KEEP": "value",
		},
	}

	loadEnv(cfg)

	if cfg.DefaultAgent != "codex" {
		t.Fatalf("DefaultAgent = %q, want %q", cfg.DefaultAgent, "codex")
	}
	if cfg.APIAddr != "127.0.0.1:18011" {
		t.Fatalf("APIAddr = %q, want %q", cfg.APIAddr, "127.0.0.1:18011")
	}
	if got := cfg.Agents["claude"].Env["KEEP"]; got != "value" {
		t.Fatalf("agent env = %q, want preserved value", got)
	}
}
