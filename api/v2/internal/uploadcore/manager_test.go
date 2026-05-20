package uploadcore

import "testing"

func TestInitWithScopeSeparatesIdenticalFilesAcrossBusinessScopes(t *testing.T) {
	manager := NewManager(t.TempDir())

	first, err := manager.InitWithScope("group-a", "same.json", 16, "abc123", 8)
	if err != nil {
		t.Fatalf("InitWithScope(group-a) returned error: %v", err)
	}
	second, err := manager.InitWithScope("group-b", "same.json", 16, "abc123", 8)
	if err != nil {
		t.Fatalf("InitWithScope(group-b) returned error: %v", err)
	}

	if first.SessionID == second.SessionID {
		t.Fatalf("session IDs are identical across scopes: %q", first.SessionID)
	}
	if first.Scope != "group-a" || second.Scope != "group-b" {
		t.Fatalf("unexpected scopes: first=%q second=%q", first.Scope, second.Scope)
	}
}
