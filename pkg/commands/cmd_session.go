package commands

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func sessionCommand() Definition {
	return Definition{
		Name:        "session",
		Description: "Manage chat sessions",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.Session == nil {
				return req.Reply(unavailableMsg)
			}
			key, isOverride := rt.Session.GetActiveKey(req.SenderID)
			if isOverride {
				return req.Reply(fmt.Sprintf("Active session: %s (override)", key))
			}
			return req.Reply(fmt.Sprintf("Active session: %s", key))
		},
		SubCommands: []SubCommand{
			{
				Name:        "list",
				Description: "List your sessions",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					if rt == nil || rt.Session == nil {
						return req.Reply(unavailableMsg)
					}
					sessions := rt.Session.ListByPeer(req.SenderID)
					if len(sessions) == 0 {
						return req.Reply("No sessions found")
					}
					activeKey, _ := rt.Session.GetActiveKey(req.SenderID)
					var sb strings.Builder
					sb.WriteString("Your sessions:\n")
					for _, s := range sessions {
						if s.Key == activeKey {
							sb.WriteString(fmt.Sprintf("  [active] %s  (updated: %s)\n", s.Key, s.Updated.Format("2006-01-02")))
						} else {
							sb.WriteString(fmt.Sprintf("           %s  (updated: %s)\n", s.Key, s.Updated.Format("2006-01-02")))
						}
					}
					return req.Reply(strings.TrimRight(sb.String(), "\n"))
				},
			},
			{
				Name:        "new",
				Description: "Create a new named session",
				ArgsUsage:   "[label]",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					if rt == nil || rt.Session == nil {
						return req.Reply(unavailableMsg)
					}
					// Collect all tokens after "/session new" as the label.
					parts := strings.Fields(req.Text)
					var label string
					if len(parts) > 2 {
						label = sanitizeSessionLabel(strings.Join(parts[2:], " "))
					}
					key, err := rt.Session.New(req.SenderID, label)
					if err != nil {
						return req.Reply(fmt.Sprintf("Failed to create session: %v", err))
					}
					return req.Reply(fmt.Sprintf("Created and switched to session: %s", key))
				},
			},
			{
				Name:        "resume",
				Description: "Resume a previous session",
				ArgsUsage:   "<key>",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					if rt == nil || rt.Session == nil {
						return req.Reply(unavailableMsg)
					}
					key := nthToken(req.Text, 2)
					if key == "" {
						return req.Reply("Usage: /session resume <key>")
					}
					if err := rt.Session.Resume(req.SenderID, key); err != nil {
						return req.Reply(fmt.Sprintf("Failed to resume session: %v", err))
					}
					return req.Reply(fmt.Sprintf("Switched to session: %s", key))
				},
			},
			{
				Name:        "delete",
				Description: "Delete a session",
				ArgsUsage:   "<key>",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					if rt == nil || rt.Session == nil {
						return req.Reply(unavailableMsg)
					}
					key := nthToken(req.Text, 2)
					if key == "" {
						return req.Reply("Usage: /session delete <key>")
					}
					wasActive, err := rt.Session.Delete(req.SenderID, key)
					if err != nil {
						return req.Reply(fmt.Sprintf("Failed to delete session: %v", err))
					}
					resp := fmt.Sprintf("Deleted session: %s", key)
					if wasActive {
						resp += "\nWarning: this was your active session. Switched back to automatic routing."
					}
					return req.Reply(resp)
				},
			},
			{
				Name:        "reset",
				Description: "Switch to automatic session routing",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					if rt == nil || rt.Session == nil {
						return req.Reply(unavailableMsg)
					}
					if err := rt.Session.Reset(req.SenderID); err != nil {
						return req.Reply(fmt.Sprintf("Failed to clear session override: %v", err))
					}
					return req.Reply("Switched to automatic session routing")
				},
			},
		},
	}
}

// sanitizeSessionLabel lowercases the label, replaces spaces with hyphens, and
// removes any characters outside [a-z0-9-].
func sanitizeSessionLabel(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	label = strings.ReplaceAll(label, " ", "-")
	var b strings.Builder
	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	result := b.String()
	if result == "" {
		result = time.Now().Format("20060102-150405")
	}
	return result
}
