package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fastclaw-ai/weclaw/agent"
	"github.com/fastclaw-ai/weclaw/ilink"
	"github.com/fastclaw-ai/weclaw/messaging"
)

func TestAuthorizeServiceToken(t *testing.T) {
	server := NewServer(nil, "", []ServicePolicy{{
		Name:         "movies",
		Token:        "secret",
		AllowedUsers: []string{"user-1"},
	}})
	request := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	request.Header.Set("Authorization", "Bearer secret")

	policy, ok := server.authorize(request.Header.Get("Authorization"))
	if !ok || policy.Name != "movies" {
		t.Fatalf("policy = %#v, ok = %v", policy, ok)
	}
}

func TestAuthorizeRejectsUnknownToken(t *testing.T) {
	server := NewServer(nil, "", []ServicePolicy{{
		Name:  "movies",
		Token: "secret",
	}})
	if _, ok := server.authorize("Bearer wrong"); ok {
		t.Fatal("unknown token was accepted")
	}
}

func TestAllowedUsersDefaultDeny(t *testing.T) {
	if isAllowed(nil, "user-1") {
		t.Fatal("empty allowlist must deny")
	}
	if !isAllowed([]string{"*"}, "user-1") {
		t.Fatal("wildcard allowlist must allow")
	}
}

func TestContextTokenUsesAccountAndUser(t *testing.T) {
	store := messaging.NewContextTokenStore()
	store.Store("account-a", "user-1", "token-a")
	store.Store("account-b", "user-1", "token-b")
	server := NewServer(nil, "", nil)
	server.SetContextTokenStore(store)

	got, ok := server.contextToken("account-b", "user-1")
	if !ok || got != "token-b" {
		t.Fatalf("contextToken() = %q, %v; want token-b, true", got, ok)
	}
	if _, ok := server.contextToken("account-a", "missing"); ok {
		t.Fatal("missing user unexpectedly returned a context token")
	}
}

func TestNativeSendRejectsMissingContextToken(t *testing.T) {
	server := NewServer(nil, "", []ServicePolicy{{
		Name:         "movies",
		Token:        "secret",
		AllowedUsers: []string{"user-1"},
	}})
	server.AddClient(ilink.NewClient(&ilink.Credentials{
		ILinkBotID: "account-a",
	}))
	server.SetContextTokenStore(messaging.NewContextTokenStore())
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/messages",
		strings.NewReader(
			`{"provider_instance_id":"account-a","to":"user-1",`+
				`"messages":[{"type":"text","text":"complete"}]}`,
		),
	)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()

	server.handleNativeSend(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf(
			"status = %d, body = %q; want 409",
			response.Code,
			response.Body.String(),
		)
	}
}

func TestPrepareMessagesRejectsUnavailableMedia(t *testing.T) {
	mediaServer := httptest.NewServer(http.NotFoundHandler())
	defer mediaServer.Close()

	prepared, err := prepareMessages(context.Background(), []agent.OutboundMessage{
		{Type: "text", Text: "authentication required"},
		{Type: "image", MediaURL: mediaServer.URL + "/expired.png"},
	})
	if err == nil {
		t.Fatal("prepareMessages() succeeded for unavailable media")
	}
	if prepared != nil {
		t.Fatalf("prepared = %#v; want nil before delivery begins", prepared)
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("error = %q; want HTTP 404", err)
	}
}

func TestPrepareMessagesDownloadsEveryAttachment(t *testing.T) {
	requests := 0
	mediaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png"))
	}))
	defer mediaServer.Close()

	prepared, err := prepareMessages(context.Background(), []agent.OutboundMessage{
		{Type: "text", Text: "authentication required"},
		{Type: "image", MediaURL: mediaServer.URL + "/qr.png"},
	})
	if err != nil {
		t.Fatalf("prepareMessages() error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("media requests = %d; want 1", requests)
	}
	if len(prepared) != 2 || prepared[1].media == nil {
		t.Fatalf("prepared = %#v; want staged media", prepared)
	}
}
