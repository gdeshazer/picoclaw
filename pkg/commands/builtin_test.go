package commands

import (
	"context"
	"strings"
	"testing"
	"time"
)

func findDefinitionByName(t *testing.T, defs []Definition, name string) Definition {
	t.Helper()
	for _, def := range defs {
		if def.Name == name {
			return def
		}
	}
	t.Fatalf("missing /%s definition", name)
	return Definition{}
}

func TestBuiltinHelpHandler_ReturnsFormattedMessage(t *testing.T) {
	defs := BuiltinDefinitions()
	helpDef := findDefinitionByName(t, defs, "help")
	if helpDef.Handler == nil {
		t.Fatalf("/help handler should not be nil")
	}

	var reply string
	err := helpDef.Handler(context.Background(), Request{
		Text: "/help",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	}, nil)
	if err != nil {
		t.Fatalf("/help handler error: %v", err)
	}
	// Verify header
	if !strings.Contains(reply, "Available commands:") {
		t.Fatalf("/help reply missing header, got %q", reply)
	}
	// Verify subcommands are shown indented under parent
	if !strings.Contains(reply, "/show - ") {
		t.Fatalf("/help reply missing /show, got %q", reply)
	}
	if !strings.Contains(reply, "  model - ") {
		t.Fatalf("/help reply missing indented 'model' subcommand under /show, got %q", reply)
	}
	if !strings.Contains(reply, "/list - ") {
		t.Fatalf("/help reply missing /list, got %q", reply)
	}
	if !strings.Contains(reply, "  models - ") {
		t.Fatalf("/help reply missing indented 'models' subcommand under /list, got %q", reply)
	}
	// Verify session command appears
	if !strings.Contains(reply, "/session - ") {
		t.Fatalf("/help reply missing /session, got %q", reply)
	}
}

func TestBuiltinShowChannel_PreservesUserVisibleBehavior(t *testing.T) {
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), nil)

	cases := []string{"telegram", "whatsapp"}
	for _, channel := range cases {
		var reply string
		res := ex.Execute(context.Background(), Request{
			Channel: channel,
			Text:    "/show channel",
			Reply: func(text string) error {
				reply = text
				return nil
			},
		})
		if res.Outcome != OutcomeHandled {
			t.Fatalf("/show channel on %s: outcome=%v, want=%v", channel, res.Outcome, OutcomeHandled)
		}
		want := "Current Channel: " + channel
		if reply != want {
			t.Fatalf("/show channel reply=%q, want=%q", reply, want)
		}
	}
}

func TestBuiltinListChannels_UsesGetEnabledChannels(t *testing.T) {
	rt := &Runtime{
		GetEnabledChannels: func() []string {
			return []string{"telegram", "slack"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list channels: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "telegram") || !strings.Contains(reply, "slack") {
		t.Fatalf("/list channels reply=%q, want telegram and slack", reply)
	}
}

func TestBuiltinShowAgents_RestoresOldBehavior(t *testing.T) {
	rt := &Runtime{
		ListAgentIDs: func() []string {
			return []string{"default", "coder"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/show agents",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/show agents: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "default") || !strings.Contains(reply, "coder") {
		t.Fatalf("/show agents reply=%q, want agent IDs", reply)
	}
}

func TestBuiltinListAgents_RestoresOldBehavior(t *testing.T) {
	rt := &Runtime{
		ListAgentIDs: func() []string {
			return []string{"default", "coder"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list agents",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list agents: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "default") || !strings.Contains(reply, "coder") {
		t.Fatalf("/list agents reply=%q, want agent IDs", reply)
	}
}

func TestFormatHelpMessage_ShowsSubcommandDetails(t *testing.T) {
	defs := []Definition{
		{
			Name:        "parent",
			Description: "Parent command",
			SubCommands: []SubCommand{
				{Name: "child1", Description: "First child"},
				{Name: "child2", Description: "Second child", ArgsUsage: "<arg>"},
			},
		},
	}
	out := formatHelpMessage(defs)
	if !strings.Contains(out, "/parent - Parent command") {
		t.Fatalf("missing parent line, got %q", out)
	}
	if !strings.Contains(out, "  child1 - First child") {
		t.Fatalf("missing child1 line, got %q", out)
	}
	if !strings.Contains(out, "  child2 <arg> - Second child") {
		t.Fatalf("missing child2 line with args, got %q", out)
	}
}

func TestFormatHelpMessage_SimpleCommand(t *testing.T) {
	defs := []Definition{
		{Name: "ping", Description: "Pong"},
	}
	out := formatHelpMessage(defs)
	if !strings.Contains(out, "/ping - Pong") {
		t.Fatalf("missing simple command line, got %q", out)
	}
}

func TestFormatHelpMessage_Empty(t *testing.T) {
	out := formatHelpMessage(nil)
	if out != "No commands available." {
		t.Fatalf("expected 'No commands available.', got %q", out)
	}
}

func TestSessionCommand_ShowActive(t *testing.T) {
	rt := &Runtime{
		Session: &SessionOps{
			GetActiveKey: func(string) (string, bool) {
				return "agent:main:user123", false
			},
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		SenderID: "user123",
		Text:     "/session",
		Reply:    func(text string) error { reply = text; return nil },
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want handled", res.Outcome)
	}
	if !strings.Contains(reply, "Active session: agent:main:user123") {
		t.Fatalf("unexpected reply: %q", reply)
	}
}

func TestSessionCommand_List(t *testing.T) {
	rt := &Runtime{
		Session: &SessionOps{
			GetActiveKey: func(string) (string, bool) { return "key1", false },
			ListByPeer: func(peerID string) []SessionInfo {
				return []SessionInfo{
					{Key: "key1", Updated: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
					{Key: "key2", Updated: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
				}
			},
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		SenderID: "user1",
		Text:     "/session list",
		Reply:    func(text string) error { reply = text; return nil },
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want handled", res.Outcome)
	}
	if !strings.Contains(reply, "[active] key1") {
		t.Fatalf("expected active marker on key1, got %q", reply)
	}
	if !strings.Contains(reply, "key2") {
		t.Fatalf("expected key2 in list, got %q", reply)
	}
}

func TestSessionCommand_New(t *testing.T) {
	var calledLabel string
	rt := &Runtime{
		Session: &SessionOps{
			New: func(senderID, label string) (string, error) {
				calledLabel = label
				return "auto:named:" + label, nil
			},
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	ex.Execute(context.Background(), Request{
		SenderID: "u1",
		Text:     "/session new myproject",
		Reply:    func(text string) error { reply = text; return nil },
	})
	if calledLabel != "myproject" {
		t.Fatalf("expected label 'myproject', got %q", calledLabel)
	}
	if !strings.Contains(reply, "Created and switched to session:") {
		t.Fatalf("unexpected reply: %q", reply)
	}
}

func TestSessionCommand_Resume(t *testing.T) {
	var resumedKey string
	rt := &Runtime{
		Session: &SessionOps{
			Resume: func(senderID, key string) error {
				resumedKey = key
				return nil
			},
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	ex.Execute(context.Background(), Request{
		SenderID: "u1",
		Text:     "/session resume agent:main:mykey",
		Reply:    func(text string) error { reply = text; return nil },
	})
	if resumedKey != "agent:main:mykey" {
		t.Fatalf("expected resumed key 'agent:main:mykey', got %q", resumedKey)
	}
	if !strings.Contains(reply, "Switched to session:") {
		t.Fatalf("unexpected reply: %q", reply)
	}
}

func TestSessionCommand_Delete(t *testing.T) {
	var deletedKey string
	rt := &Runtime{
		Session: &SessionOps{
			Delete: func(senderID, key string) (bool, error) {
				deletedKey = key
				return false, nil
			},
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	ex.Execute(context.Background(), Request{
		SenderID: "u1",
		Text:     "/session delete somekey",
		Reply:    func(text string) error { reply = text; return nil },
	})
	if deletedKey != "somekey" {
		t.Fatalf("expected deleted key 'somekey', got %q", deletedKey)
	}
	if !strings.Contains(reply, "Deleted session: somekey") {
		t.Fatalf("unexpected reply: %q", reply)
	}
}

func TestSessionCommand_Reset(t *testing.T) {
	var resetCalled bool
	rt := &Runtime{
		Session: &SessionOps{
			Reset: func(string) error {
				resetCalled = true
				return nil
			},
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	ex.Execute(context.Background(), Request{
		SenderID: "u1",
		Text:     "/session reset",
		Reply:    func(text string) error { reply = text; return nil },
	})
	if !resetCalled {
		t.Fatal("expected Reset callback to be called")
	}
	if !strings.Contains(reply, "automatic session routing") {
		t.Fatalf("unexpected reply: %q", reply)
	}
}

func TestSessionCommand_UnavailableWithoutRuntime(t *testing.T) {
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), nil)

	var reply string
	res := ex.Execute(context.Background(), Request{
		SenderID: "u1",
		Text:     "/session",
		Reply:    func(text string) error { reply = text; return nil },
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want handled", res.Outcome)
	}
	if reply != unavailableMsg {
		t.Fatalf("expected unavailable message, got %q", reply)
	}
}
