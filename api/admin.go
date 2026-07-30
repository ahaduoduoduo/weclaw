package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/fastclaw-ai/weclaw/config"
	"github.com/fastclaw-ai/weclaw/management"
	weclawweb "github.com/fastclaw-ai/weclaw/web"
)

const adminCookie = "weclaw_admin"

type AdminServices struct {
	Auth              *management.Auth
	Configs           *config.RuntimeStore
	Access            *management.AccessStore
	Accounts          *management.AccountManager
	Logins            *management.LoginManager
	OnAgentsChanged   func()
	OnPoliciesChanged func()
}

type agentInput struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Command        string            `json:"command"`
	Args           []string          `json:"args"`
	Aliases        []string          `json:"aliases"`
	Cwd            string            `json:"cwd"`
	Env            map[string]string `json:"env"`
	Model          string            `json:"model"`
	SystemPrompt   string            `json:"system_prompt"`
	Endpoint       string            `json:"endpoint"`
	APIKey         string            `json:"api_key"`
	Headers        map[string]string `json:"headers"`
	MaxHistory     int               `json:"max_history"`
	OutboundToken  string            `json:"outbound_token"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	Public         bool              `json:"public"`
	Default        bool              `json:"default"`
}

func (s *Server) registerAdmin(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleWeb)
	mux.HandleFunc("/styles.css", staticFile("styles.css", "text/css; charset=utf-8"))
	mux.HandleFunc("/app.js", staticFile("app.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("/api/admin/status", s.handleAdminStatus)
	mux.HandleFunc("/api/admin/setup", s.handleAdminSetup)
	mux.HandleFunc("/api/admin/login", s.handleAdminLogin)
	mux.HandleFunc("/api/admin/logout", s.handleAdminLogout)
	mux.HandleFunc("/api/admin/accounts", s.adminOnly(s.handleAccounts))
	mux.HandleFunc("/api/admin/login-sessions", s.adminOnly(s.handleLoginSessions))
	mux.HandleFunc("/api/admin/login-sessions/", s.adminOnly(s.handleLoginSession))
	mux.HandleFunc("/api/admin/agents", s.adminOnly(s.handleAgents))
	mux.HandleFunc("/api/admin/agents/", s.adminOnly(s.handleAgent))
	mux.HandleFunc("/api/admin/users", s.adminOnly(s.handleUsers))
	mux.HandleFunc("/api/admin/users/", s.adminOnly(s.handleUser))
}

func staticFile(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		data, err := weclawweb.Assets.ReadFile(name)
		if err != nil {
			http.NotFound(w, nil)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(data)
	}
}

func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	staticFile("index.html", "text/html; charset=utf-8")(w, r)
}

func (s *Server) handleAdminStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || s.admin == nil {
		http.Error(w, "not available", http.StatusNotFound)
		return
	}
	_, authenticated := s.adminToken(r)
	writeJSON(w, map[string]any{
		"setup_required": s.admin.Auth.SetupRequired(),
		"authenticated":  authenticated,
	})
}

func (s *Server) handleAdminSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || s.admin == nil {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if decodeJSON(w, r, &body) != nil {
		return
	}
	token, err := s.admin.Auth.Setup(body.Password)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	setAdminCookie(w, token)
	writeOK(w)
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || s.admin == nil {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if decodeJSON(w, r, &body) != nil {
		return
	}
	token, err := s.admin.Auth.Login(body.Password)
	if err != nil {
		writeError(w, err, http.StatusUnauthorized)
		return
	}
	setAdminCookie(w, token)
	writeOK(w)
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	token, _ := s.adminToken(r)
	s.admin.Auth.Logout(token)
	http.SetCookie(w, &http.Cookie{
		Name: adminCookie, MaxAge: -1, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
	writeOK(w)
}

func (s *Server) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.adminToken(r); !ok {
			writeJSONStatus(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) adminToken(r *http.Request) (string, bool) {
	if s.admin == nil {
		return "", false
	}
	cookie, err := r.Cookie(adminCookie)
	if err != nil || !s.admin.Auth.Valid(cookie.Value) {
		return "", false
	}
	return cookie.Value, true
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.admin.Accounts.List())
	case http.MethodDelete:
		botID := r.URL.Query().Get("bot_id")
		if err := s.admin.Accounts.Delete(botID); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		s.RemoveClient(botID)
		writeOK(w)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLoginSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	session, err := s.admin.Logins.Start()
	if err != nil {
		writeError(w, err, http.StatusBadGateway)
		return
	}
	writeJSON(w, session)
}

func (s *Server) handleLoginSession(w http.ResponseWriter, r *http.Request) {
	suffix := strings.TrimPrefix(r.URL.Path, "/api/admin/login-sessions/")
	if strings.HasSuffix(suffix, "/qr.png") {
		id := strings.TrimSuffix(suffix, "/qr.png")
		png, ok := s.admin.Logins.PNG(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
		return
	}
	session, ok := s.admin.Logins.Get(suffix)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, session)
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	agents := s.admin.Configs.Agents()
	result := make([]agentInput, 0, len(agents))
	for name, value := range agents {
		result = append(result, agentView(name, value, s.admin.Configs.DefaultAgent()))
	}
	writeJSON(w, result)
}

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/admin/agents/")
	if name == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(name, "/public") {
		name = strings.TrimSuffix(name, "/public")
		var body struct {
			Public bool `json:"public"`
		}
		if decodeJSON(w, r, &body) != nil {
			return
		}
		if err := s.admin.Access.SetPublic(name, body.Public); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		s.admin.OnPoliciesChanged()
		writeOK(w)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var body agentInput
		if decodeJSON(w, r, &body) != nil {
			return
		}
		existing, _ := s.admin.Configs.Agent(name)
		value := inputAgent(body, existing)
		if err := s.admin.Configs.SaveAgent(name, value); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		if body.Default {
			_ = s.admin.Configs.SetDefaultAgent(name)
		}
		s.admin.OnAgentsChanged()
		writeJSON(w, agentView(name, value, s.admin.Configs.DefaultAgent()))
	case http.MethodDelete:
		if err := s.admin.Configs.DeleteAgent(name); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		s.admin.OnAgentsChanged()
		writeOK(w)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	agents := s.admin.Configs.AgentNames()
	type view struct {
		management.User
		Permissions map[string]bool `json:"permissions"`
	}
	result := make([]view, 0)
	for _, user := range s.admin.Access.Users() {
		item := view{User: user, Permissions: make(map[string]bool)}
		for _, name := range agents {
			item.Permissions[name] = s.admin.Access.Allowed(user.AccountID, user.UserID, name)
		}
		result = append(result, item)
	}
	writeJSON(w, result)
}

func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	accountID, userID := parts[0], parts[1]
	if len(parts) == 3 && parts[2] == "permissions" {
		var body struct {
			Agent   string `json:"agent"`
			Allowed bool   `json:"allowed"`
		}
		if decodeJSON(w, r, &body) != nil {
			return
		}
		if err := s.admin.Access.SetPermission(accountID, userID, body.Agent, body.Allowed); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		s.admin.OnPoliciesChanged()
		writeOK(w)
		return
	}
	var body struct {
		Status       string `json:"status"`
		DefaultAgent string `json:"default_agent"`
	}
	if decodeJSON(w, r, &body) != nil {
		return
	}
	if err := s.admin.Access.UpdateUser(accountID, userID, body.Status, body.DefaultAgent); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeOK(w)
}

func agentView(name string, value config.AgentConfig, defaultName string) agentInput {
	public := false
	for _, allowed := range value.AllowedUsers {
		public = public || allowed == "*"
	}
	return agentInput{
		Name: name, Type: value.Type, Command: value.Command, Args: value.Args,
		Aliases: value.Aliases, Cwd: value.Cwd, Env: value.Env, Model: value.Model,
		SystemPrompt: value.SystemPrompt, Endpoint: value.Endpoint, Headers: value.Headers,
		MaxHistory: value.MaxHistory, TimeoutSeconds: value.TimeoutSeconds,
		Public: public, Default: name == defaultName,
	}
}

func inputAgent(body agentInput, existing config.AgentConfig) config.AgentConfig {
	value := config.AgentConfig{
		Type: body.Type, Command: body.Command, Args: body.Args, Aliases: body.Aliases,
		Cwd: body.Cwd, Env: body.Env, Model: body.Model, SystemPrompt: body.SystemPrompt,
		Endpoint: body.Endpoint, Headers: body.Headers, MaxHistory: body.MaxHistory,
		TimeoutSeconds: body.TimeoutSeconds, AllowedUsers: existing.AllowedUsers,
		APIKey: existing.APIKey, OutboundToken: existing.OutboundToken,
	}
	if body.APIKey != "" {
		value.APIKey = body.APIKey
	}
	if body.OutboundToken != "" {
		value.OutboundToken = body.OutboundToken
	}
	if body.Public {
		found := false
		for _, item := range value.AllowedUsers {
			found = found || item == "*"
		}
		if !found {
			value.AllowedUsers = append([]string{"*"}, value.AllowedUsers...)
		}
	} else {
		next := value.AllowedUsers[:0]
		for _, item := range value.AllowedUsers {
			if item != "*" {
				next = append(next, item)
			}
		}
		value.AllowedUsers = next
	}
	return value
}

func setAdminCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: adminCookie, Value: token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteStrictMode, Expires: time.Now().Add(24 * time.Hour),
	})
}

func writeError(w http.ResponseWriter, err error, status int) {
	writeJSONStatus(w, map[string]string{"error": err.Error()}, status)
}

func writeJSON(w http.ResponseWriter, value any) {
	writeJSONStatus(w, value, http.StatusOK)
}

func writeJSONStatus(w http.ResponseWriter, value any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
