package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
