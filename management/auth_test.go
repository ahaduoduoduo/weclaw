package management

import "testing"

func TestAuthSetupLoginAndLogout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth, err := NewAuth()
	if err != nil {
		t.Fatal(err)
	}
	if !auth.SetupRequired() {
		t.Fatal("new auth store should require setup")
	}
	token, err := auth.Setup("a-long-test-password")
	if err != nil || !auth.Valid(token) {
		t.Fatalf("setup token invalid: %v", err)
	}
	auth.Logout(token)
	if auth.Valid(token) {
		t.Fatal("logged out token remained valid")
	}
	if _, err := auth.Login("wrong-password"); err == nil {
		t.Fatal("wrong password was accepted")
	}
	if token, err = auth.Login("a-long-test-password"); err != nil || !auth.Valid(token) {
		t.Fatalf("login failed: %v", err)
	}
}
