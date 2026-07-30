package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fastclaw-ai/weclaw/agent"
	"github.com/fastclaw-ai/weclaw/ilink"
	"github.com/fastclaw-ai/weclaw/messaging"
)

const maxRequestBody = 8 << 20

// ServicePolicy authorizes proactive messages from one native service.
type ServicePolicy struct {
	Name         string
	Token        string
	AllowedUsers []string
}

// Server provides legacy and channel-neutral outbound message APIs.
type Server struct {
	mu       sync.RWMutex
	clients  []*ilink.Client
	byBotID  map[string]*ilink.Client
	addr     string
	policies []ServicePolicy
	admin    *AdminServices
}

func (s *Server) SetAdmin(admin *AdminServices) {
	s.admin = admin
}

func (s *Server) SetPolicies(policies []ServicePolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies = append([]ServicePolicy(nil), policies...)
}

func (s *Server) AddClient(client *ilink.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byBotID[client.BotID()]; !exists {
		s.clients = append(s.clients, client)
	}
	s.byBotID[client.BotID()] = client
}

func (s *Server) RemoveClient(botID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byBotID, botID)
	next := s.clients[:0]
	for _, client := range s.clients {
		if client.BotID() != botID {
			next = append(next, client)
		}
	}
	s.clients = next
}

// NewServer creates an API server.
func NewServer(
	clients []*ilink.Client,
	addr string,
	policies []ServicePolicy,
) *Server {
	if addr == "" {
		addr = "127.0.0.1:18011"
	}
	byBotID := make(map[string]*ilink.Client, len(clients))
	for _, client := range clients {
		byBotID[client.BotID()] = client
	}
	return &Server{
		clients:  clients,
		byBotID:  byBotID,
		addr:     addr,
		policies: policies,
	}
}

// NativeSendRequest is the channel-neutral proactive message request.
type NativeSendRequest struct {
	ProviderInstanceID string                  `json:"provider_instance_id,omitempty"`
	To                 string                  `json:"to"`
	Messages           []agent.OutboundMessage `json:"messages"`
}

// Run starts the HTTP server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/messages", s.handleNativeSend)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	if s.admin != nil {
		s.registerAdmin(mux)
	}

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	log.Printf("[api] listening on %s", s.addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) handleNativeSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	policy, ok := s.authorize(r.Header.Get("Authorization"))
	if !ok {
		http.Error(w, "invalid service token", http.StatusUnauthorized)
		return
	}

	var req NativeSendRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if req.To == "" || len(req.Messages) == 0 {
		http.Error(w, `"to" and "messages" are required`, http.StatusBadRequest)
		return
	}
	if !isAllowed(policy.AllowedUsers, req.To, req.ProviderInstanceID) {
		http.Error(w, "target is not allowed for this service", http.StatusForbidden)
		return
	}

	client, err := s.selectClient(req.ProviderInstanceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if err := sendMessages(r.Context(), client, req.To, req.Messages); err != nil {
		log.Printf("[api] native send failed for %s: %v", policy.Name, err)
		http.Error(w, "send failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeOK(w)
}

func (s *Server) authorize(header string) (ServicePolicy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return ServicePolicy{}, false
	}
	for _, policy := range s.policies {
		if len(token) == len(policy.Token) &&
			subtle.ConstantTimeCompare([]byte(token), []byte(policy.Token)) == 1 {
			return policy, true
		}
	}
	return ServicePolicy{}, false
}

func (s *Server) selectClient(providerInstanceID string) (*ilink.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if providerInstanceID != "" {
		client := s.byBotID[providerInstanceID]
		if client == nil {
			return nil, fmt.Errorf("unknown provider_instance_id")
		}
		return client, nil
	}
	if len(s.clients) == 0 {
		return nil, fmt.Errorf("no accounts configured")
	}
	return s.clients[0], nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return err
	}
	return nil
}

func sendMessages(
	ctx context.Context,
	client *ilink.Client,
	to string,
	messages []agent.OutboundMessage,
) error {
	for _, message := range messages {
		if message.Text != "" {
			if err := messaging.SendTextReply(ctx, client, to, message.Text, "", ""); err != nil {
				return err
			}
		}
		if message.MediaURL != "" {
			if err := messaging.SendMediaFromURL(ctx, client, to, message.MediaURL, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

func isAllowed(users []string, userID string, accountID ...string) bool {
	if slices.Contains(users, "*") || slices.Contains(users, userID) {
		return true
	}
	return len(accountID) > 0 &&
		slices.Contains(users, accountID[0]+":"+userID)
}

func writeOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
