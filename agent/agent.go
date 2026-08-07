package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Attachment is a channel-neutral inbound media item.
type Attachment struct {
	Type        string `json:"type"`
	FileName    string `json:"file_name,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	DataBase64  string `json:"data_base64,omitempty"`
	URL         string `json:"url,omitempty"`
}

// InboundMessage is the stable message envelope used by native HTTP services.
type InboundMessage struct {
	Version            string       `json:"version"`
	EventID            string       `json:"event_id"`
	EventType          string       `json:"event_type"`
	Provider           string       `json:"provider"`
	ProviderInstanceID string       `json:"provider_instance_id"`
	ConversationID     string       `json:"conversation_id"`
	SenderID           string       `json:"sender_id"`
	MessageID          string       `json:"message_id"`
	MessageType        string       `json:"message_type"`
	Text               string       `json:"text,omitempty"`
	Attachments        []Attachment `json:"attachments,omitempty"`
	Timestamp          time.Time    `json:"timestamp"`
	Capabilities       []string     `json:"capabilities,omitempty"`
}

// OutboundMessage is a channel-neutral reply returned by a native service.
type OutboundMessage struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	MediaURL  string `json:"media_url,omitempty"`
	FileName  string `json:"file_name,omitempty"`
	ReplyToID string `json:"reply_to_id,omitempty"`
}

// MessageAgent handles complete channel messages and can return structured replies.
type MessageAgent interface {
	HandleMessage(ctx context.Context, message InboundMessage) ([]OutboundMessage, error)
}

// ProviderSessionResetter resets a conversation that belongs to a specific
// messaging provider instance. Native services use this to distinguish the
// same user or conversation ID across multiple signed-in accounts.
type ProviderSessionResetter interface {
	ResetProviderSession(
		ctx context.Context,
		providerInstanceID string,
		conversationID string,
	) (string, error)
}

// AgentInfo holds metadata about an agent for logging/debugging.
type AgentInfo struct {
	Name    string // e.g. "claude-acp", "claude", "gpt-4o"
	Type    string // e.g. "acp", "cli", "http"
	Model   string // e.g. "sonnet", "gpt-4o-mini"
	Command string // binary path, e.g. "/usr/local/bin/claude-agent-acp"
	PID     int    // subprocess PID (0 if not applicable, e.g. http agent)
}

// String returns a human-readable summary for logging.
func (i AgentInfo) String() string {
	s := fmt.Sprintf("name=%s, type=%s, model=%s, command=%s", i.Name, i.Type, i.Model, i.Command)
	if i.PID > 0 {
		s += fmt.Sprintf(", pid=%d", i.PID)
	}
	return s
}

// defaultWorkspace returns ~/.weclaw/workspace as the default working directory.
func defaultWorkspace() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}
	dir := filepath.Join(home, ".weclaw", "workspace")
	os.MkdirAll(dir, 0o755)
	return dir
}

// mergeEnv merges extra environment variables into the base environment.
func mergeEnv(base []string, extra map[string]string) ([]string, error) {
	if len(extra) == 0 {
		return base, nil
	}

	merged := append([]string(nil), base...)
	indexByKey := make(map[string]int, len(base))
	for i, entry := range merged {
		key, _, found := strings.Cut(entry, "=")
		if !found || key == "" {
			continue
		}
		indexByKey[key] = i
	}

	newKeys := make([]string, 0, len(extra))
	for key, value := range extra {
		if key == "" || strings.Contains(key, "=") {
			return nil, fmt.Errorf("invalid env key %q", key)
		}
		entry := key + "=" + value
		if idx, ok := indexByKey[key]; ok {
			merged[idx] = entry
			continue
		}
		newKeys = append(newKeys, key)
	}

	sort.Strings(newKeys)
	for _, key := range newKeys {
		merged = append(merged, key+"="+extra[key])
	}

	return merged, nil
}

// Agent is the interface for AI chat agents.
type Agent interface {
	// Chat sends a message to the agent and returns the response.
	// conversationID is used to maintain conversation history per user.
	Chat(ctx context.Context, conversationID string, message string) (string, error)

	// ResetSession clears the existing session for the given conversationID and
	// starts a new one. Returns the new session ID if immediately available
	// (ACP mode), or an empty string if the ID will be assigned on next Chat
	// (CLI mode) or is not applicable (HTTP mode).
	ResetSession(ctx context.Context, conversationID string) (string, error)

	// Info returns metadata about this agent.
	Info() AgentInfo

	// SetCwd changes the working directory for subsequent operations.
	SetCwd(cwd string)
}
