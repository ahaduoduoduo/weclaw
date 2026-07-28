package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNativeServiceHandleMessage(t *testing.T) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer inbound-secret" {
			t.Fatalf("Authorization = %q", got)
		}
		var message InboundMessage
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if message.Provider != "wechat" || message.SenderID != "user-1" {
			t.Fatalf("unexpected message: %#v", message)
		}
		_ = json.NewEncoder(w).Encode(NativeServiceResponse{
			Messages: []OutboundMessage{{
				Type: "text",
				Text: "accepted",
			}},
		})
	}))
	defer server.Close()

	service := NewNativeService(NativeServiceConfig{
		Endpoint:     server.URL,
		APIKey:       "inbound-secret",
		AllowedUsers: []string{"user-1"},
	})
	replies, err := service.HandleMessage(context.Background(), InboundMessage{
		Version:        "2026-07-01",
		Provider:       "wechat",
		SenderID:       "user-1",
		ConversationID: "user-1",
		Timestamp:      time.Now(),
	})
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(replies) != 1 || replies[0].Text != "accepted" {
		t.Fatalf("replies = %#v", replies)
	}
}

func TestNativeServiceRejectsUnknownSender(t *testing.T) {
	service := NewNativeService(NativeServiceConfig{
		Endpoint:     "http://127.0.0.1:1",
		AllowedUsers: []string{"user-1"},
	})
	_, err := service.HandleMessage(context.Background(), InboundMessage{
		SenderID: "user-2",
	})
	if err == nil {
		t.Fatal("expected unknown sender to be rejected")
	}
}

func TestNativeServiceAllowsWildcard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	service := NewNativeService(NativeServiceConfig{
		Endpoint:     server.URL,
		AllowedUsers: []string{"*"},
	})
	if _, err := service.HandleMessage(context.Background(), InboundMessage{
		SenderID: "anyone",
	}); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
}
