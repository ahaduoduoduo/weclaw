package management

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fastclaw-ai/weclaw/ilink"
	"github.com/fastclaw-ai/weclaw/messaging"
)

type Account struct {
	BotID  string `json:"bot_id"`
	Online bool   `json:"online"`
}

type AccountManager struct {
	ctx      context.Context
	handler  *messaging.Handler
	onClient func(*ilink.Client)
	mu       sync.Mutex
	cancels  map[string]context.CancelFunc
}

func NewAccountManager(
	ctx context.Context,
	handler *messaging.Handler,
	onClient func(*ilink.Client),
) *AccountManager {
	return &AccountManager{
		ctx: ctx, handler: handler, onClient: onClient,
		cancels: make(map[string]context.CancelFunc),
	}
}

func (m *AccountManager) Load() error {
	accounts, err := ilink.LoadAllCredentials()
	if err != nil {
		return err
	}
	for _, creds := range accounts {
		m.Add(creds)
	}
	return nil
}

func (m *AccountManager) Add(creds *ilink.Credentials) {
	m.mu.Lock()
	if _, exists := m.cancels[creds.ILinkBotID]; exists {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(m.ctx)
	m.cancels[creds.ILinkBotID] = cancel
	m.mu.Unlock()
	client := ilink.NewClient(creds)
	m.onClient(client)
	go m.monitor(ctx, client)
}

func (m *AccountManager) List() []Account {
	creds, _ := ilink.LoadAllCredentials()
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Account, 0, len(creds))
	for _, item := range creds {
		_, online := m.cancels[item.ILinkBotID]
		result = append(result, Account{BotID: item.ILinkBotID, Online: online})
	}
	return result
}

func (m *AccountManager) Delete(botID string) error {
	m.mu.Lock()
	if cancel := m.cancels[botID]; cancel != nil {
		cancel()
		delete(m.cancels, botID)
	}
	m.mu.Unlock()
	return ilink.DeleteCredentials(botID)
}

func (m *AccountManager) monitor(ctx context.Context, client *ilink.Client) {
	delay := 3 * time.Second
	for ctx.Err() == nil {
		monitor, err := ilink.NewMonitor(client, m.handler.HandleMessage)
		if err == nil {
			err = monitor.Run(ctx)
		}
		if ctx.Err() != nil {
			return
		}
		fmt.Printf("[account] monitor %s stopped: %v\n", client.BotID(), err)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
		if delay < 30*time.Second {
			delay *= 2
		}
	}
}
