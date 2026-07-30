package config

import (
	"fmt"
	"regexp"
	"sort"
	"sync"
)

var agentNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// RuntimeStore provides synchronized, durable access to the live config.
type RuntimeStore struct {
	mu  sync.RWMutex
	cfg *Config
}

func NewRuntimeStore(cfg *Config) *RuntimeStore {
	if cfg.Agents == nil {
		cfg.Agents = make(map[string]AgentConfig)
	}
	return &RuntimeStore{cfg: cfg}
}

func (s *RuntimeStore) Agent(name string) (AgentConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.cfg.Agents[name]
	return cloneAgent(value), ok
}

func (s *RuntimeStore) Agents() map[string]AgentConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]AgentConfig, len(s.cfg.Agents))
	for name, value := range s.cfg.Agents {
		result[name] = cloneAgent(value)
	}
	return result
}

func (s *RuntimeStore) AgentNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.cfg.Agents))
	for name := range s.cfg.Agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *RuntimeStore) DefaultAgent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.DefaultAgent
}

func (s *RuntimeStore) SaveAgent(name string, value AgentConfig) error {
	if !agentNamePattern.MatchString(name) {
		return fmt.Errorf("agent name may contain only letters, numbers, _ and -")
	}
	switch value.Type {
	case "native", "http":
		if value.Endpoint == "" {
			return fmt.Errorf("%s agent endpoint is required", value.Type)
		}
	case "acp", "cli":
		if value.Command == "" {
			return fmt.Errorf("%s agent command is required", value.Type)
		}
	default:
		return fmt.Errorf("unsupported agent type %q", value.Type)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Agents[name] = cloneAgent(value)
	if s.cfg.DefaultAgent == "" {
		s.cfg.DefaultAgent = name
	}
	return Save(s.cfg)
}

func (s *RuntimeStore) DeleteAgent(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == s.cfg.DefaultAgent {
		return fmt.Errorf("default agent cannot be deleted")
	}
	delete(s.cfg.Agents, name)
	return Save(s.cfg)
}

func (s *RuntimeStore) SetDefaultAgent(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cfg.Agents[name]; !ok {
		return fmt.Errorf("agent %q does not exist", name)
	}
	s.cfg.DefaultAgent = name
	return Save(s.cfg)
}

func (s *RuntimeStore) Aliases() map[string]string {
	return BuildAliasMap(s.Agents())
}

func cloneAgent(value AgentConfig) AgentConfig {
	value.Args = append([]string(nil), value.Args...)
	value.Aliases = append([]string(nil), value.Aliases...)
	value.AllowedUsers = append([]string(nil), value.AllowedUsers...)
	value.Env = cloneMap(value.Env)
	value.Headers = cloneMap(value.Headers)
	return value
}

func cloneMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
