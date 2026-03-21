package commands

import (
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// Runtime provides runtime dependencies to command handlers. It is constructed
// per-request by the agent loop so that per-request state (like session scope)
// can coexist with long-lived callbacks (like GetModelInfo).
type Runtime struct {
	Config             *config.Config
	GetModelInfo       func() (name, provider string)
	ListAgentIDs       func() []string
	ListDefinitions    func() []Definition
	GetEnabledChannels func() []string
	SwitchModel        func(value string) (oldModel string, err error)
	SwitchChannel      func(value string) error
	ClearHistory       func() error
	ReloadConfig       func() error
	Session            *SessionOps
}

// SessionOps provides session management callbacks.
type SessionOps struct {
	GetActiveKey func(senderID string) (key string, isOverride bool)
	ListByPeer   func(peerID string) []SessionInfo
	New          func(senderID, label string) (string, error)
	Resume       func(senderID, key string) error
	Delete       func(senderID, key string) (wasActive bool, err error)
	Reset        func(senderID string) error
}

// SessionInfo is a minimal session descriptor for command display.
type SessionInfo struct {
	Key     string
	Updated time.Time
}
