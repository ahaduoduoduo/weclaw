package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
