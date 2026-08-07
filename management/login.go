package management

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fastclaw-ai/weclaw/ilink"
	"github.com/google/uuid"
	"rsc.io/qr"
)

type LoginSession struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	BotID     string    `json:"bot_id,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	png       []byte
}

type LoginManager struct {
	mu       sync.RWMutex
	sessions map[string]*LoginSession
	accounts *AccountManager
	ctx      context.Context
}

func NewLoginManager(ctx context.Context, accounts *AccountManager) *LoginManager {
	return &LoginManager{
		sessions: make(map[string]*LoginSession),
		accounts: accounts,
		ctx:      ctx,
	}
}

func (m *LoginManager) Start() (*LoginSession, error) {
	fetchCtx, cancel := context.WithTimeout(m.ctx, 45*time.Second)
	defer cancel()
	response, err := ilink.FetchQRCode(fetchCtx)
	if err != nil {
		return nil, err
	}
	code, err := qr.Encode(response.QRCodeImgContent, qr.M)
	if err != nil {
		return nil, fmt.Errorf("encode QR image: %w", err)
	}
	session := &LoginSession{
		ID: uuid.NewString(), Status: "waiting",
		CreatedAt: time.Now().UTC(), png: code.PNG(),
	}
	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()
	go m.poll(m.ctx, session.ID, response.QRCode)
	copy := *session
	return &copy, nil
}

func (m *LoginManager) Get(id string) (LoginSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	if !ok {
		return LoginSession{}, false
	}
	copy := *session
	copy.png = nil
	return copy, true
}

func (m *LoginManager) PNG(id string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), session.png...), true
}

func (m *LoginManager) poll(ctx context.Context, id, code string) {
	creds, err := ilink.PollQRStatus(ctx, code, func(status string) {
		m.update(id, func(session *LoginSession) {
			switch status {
			case "scaned":
				session.Status = "scanned"
			case "confirmed":
				session.Status = "confirmed"
			case "expired":
				session.Status = "expired"
			}
		})
	})
	if err != nil {
		m.update(id, func(session *LoginSession) {
			if session.Status != "expired" {
				session.Status = "failed"
				session.Error = err.Error()
			}
		})
		return
	}
	if err := ilink.SaveCredentials(creds); err != nil {
		m.update(id, func(session *LoginSession) {
			session.Status = "failed"
			session.Error = err.Error()
		})
		return
	}
	m.accounts.Add(creds)
	m.update(id, func(session *LoginSession) {
		session.Status = "connected"
		session.BotID = creds.ILinkBotID
	})
}

func (m *LoginManager) update(id string, mutate func(*LoginSession)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session := m.sessions[id]; session != nil {
		mutate(session)
	}
}
