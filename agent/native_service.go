package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

// NativeService implements WeClaw's channel-neutral HTTP service protocol.
type NativeService struct {
	endpoint     string
	apiKey       string
	headers      map[string]string
	allowedUsers []string
	httpClient   *http.Client
}

// NativeServiceConfig configures one native HTTP service.
type NativeServiceConfig struct {
	Endpoint       string
	APIKey         string
	Headers        map[string]string
	AllowedUsers   []string
	TimeoutSeconds int
}

// NativeServiceResponse is returned by a native service endpoint.
type NativeServiceResponse struct {
	Messages []OutboundMessage `json:"messages"`
}

// NewNativeService creates a native HTTP service client.
func NewNativeService(cfg NativeServiceConfig) *NativeService {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &NativeService{
		endpoint:     cfg.Endpoint,
		apiKey:       cfg.APIKey,
		headers:      cfg.Headers,
		allowedUsers: append([]string(nil), cfg.AllowedUsers...),
		httpClient:   &http.Client{Timeout: timeout},
	}
}

// Info returns native service metadata.
func (s *NativeService) Info() AgentInfo {
	return AgentInfo{
		Name:    "native",
		Type:    "native",
		Command: s.endpoint,
	}
}

// SetCwd is a no-op because native services do not share a local workspace.
func (s *NativeService) SetCwd(_ string) {}

// Chat preserves compatibility with the basic Agent interface.
func (s *NativeService) Chat(ctx context.Context, conversationID, message string) (string, error) {
	replies, err := s.HandleMessage(ctx, InboundMessage{
		Version:        "2026-07-01",
		EventID:        fmt.Sprintf("compat:%s:%d", conversationID, time.Now().UnixNano()),
		EventType:      "message.created",
		Provider:       "wechat",
		ConversationID: conversationID,
		SenderID:       conversationID,
		MessageType:    "text",
		Text:           message,
		Timestamp:      time.Now().UTC(),
	})
	if err != nil {
		return "", err
	}
	var texts []string
	for _, reply := range replies {
		if reply.Text != "" {
			texts = append(texts, reply.Text)
		}
	}
	return strings.Join(texts, "\n"), nil
}

// ResetSession sends a channel-neutral reset event.
func (s *NativeService) ResetSession(ctx context.Context, conversationID string) (string, error) {
	return s.ResetProviderSession(ctx, "wechat", conversationID)
}

// ResetProviderSession sends a channel-neutral reset event for one signed-in
// provider account. The provider instance is required by native services to
// select the correct configured channel.
func (s *NativeService) ResetProviderSession(
	ctx context.Context,
	providerInstanceID string,
	conversationID string,
) (string, error) {
	now := time.Now().UTC()
	messageID := "reset:" + fmt.Sprint(now.UnixNano())
	_, err := s.HandleMessage(ctx, InboundMessage{
		Version:            "2026-07-01",
		EventID:            providerInstanceID + ":" + conversationID + ":" + messageID,
		EventType:          "conversation.reset",
		Provider:           "wechat",
		ProviderInstanceID: providerInstanceID,
		ConversationID:     conversationID,
		SenderID:           conversationID,
		MessageID:          messageID,
		MessageType:        "control",
		Timestamp:          now,
		Capabilities:       []string{"conversation.reset"},
	})
	return "", err
}

// HandleMessage delivers one normalized event to the configured service.
func (s *NativeService) HandleMessage(
	ctx context.Context,
	message InboundMessage,
) ([]OutboundMessage, error) {
	if !slices.Contains(s.allowedUsers, "*") &&
		!slices.Contains(s.allowedUsers, message.SenderID) {
		return nil, fmt.Errorf("sender %q is not allowed for this service", message.SenderID)
	}

	data, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("marshal native message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create native request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	for key, value := range s.headers {
		req.Header.Set(key, value)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("native service request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read native response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("native service HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, nil
	}

	var result NativeServiceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse native response: %w", err)
	}
	return result.Messages, nil
}
