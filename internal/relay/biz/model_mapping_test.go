package biz

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewModelMapper_EmptyPath(t *testing.T) {
	m, err := NewModelMapper("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Resolve("gpt-4o") != "gpt-4o" {
		t.Errorf("expected passthrough, got %s", m.Resolve("gpt-4o"))
	}
}

func TestModelMapper_Resolve(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "models.yaml")
	content := `{
  "models": {
    "gpt-4o": {"actual_name": "gpt-4o-2024-08-06", "capabilities": ["streaming"]},
    "claude-3-5-sonnet": {"actual_name": "claude-3-5-sonnet-20241022", "capabilities": ["vision", "streaming"]}
  }
}`
	if err := os.WriteFile(cfg, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := NewModelMapper(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := m.Resolve("gpt-4o"); got != "gpt-4o-2024-08-06" {
		t.Errorf("Resolve(gpt-4o) = %s, want gpt-4o-2024-08-06", got)
	}
	if got := m.Resolve("unknown-model"); got != "unknown-model" {
		t.Errorf("Resolve(unknown-model) = %s, want unknown-model", got)
	}
}

func TestModelMapper_HasCapability(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "models.yaml")
	content := `{
  "models": {
    "gpt-4o": {"actual_name": "gpt-4o-2024-08-06", "capabilities": ["function_call", "vision", "streaming"]}
  }
}`
	if err := os.WriteFile(cfg, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := NewModelMapper(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !m.HasCapability("gpt-4o", "vision") {
		t.Error("expected gpt-4o to have vision capability")
	}
	if m.HasCapability("gpt-4o", "embedding") {
		t.Error("expected gpt-4o to NOT have embedding capability")
	}
	if m.HasCapability("unknown", "streaming") {
		t.Error("expected unknown model to have no capabilities")
	}
}

func TestNewModelMapper_Validation_EmptyActualName(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "models.yaml")
	content := `{
  "models": {
    "gpt-4o": {"actual_name": "", "capabilities": ["streaming"]}
  }
}`
	if err := os.WriteFile(cfg, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := NewModelMapper(cfg)
	if err == nil {
		t.Fatal("expected error for empty actual_name, got nil")
	}
	if expected := "empty actual_name"; !contains(err.Error(), expected) {
		t.Errorf("error %q should contain %q", err.Error(), expected)
	}
}

func TestNewModelMapper_Validation_UnknownCapability(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "models.yaml")
	content := `{
  "models": {
    "gpt-4o": {"actual_name": "gpt-4o-2024-08-06", "capabilities": ["streaming", "telepathy"]}
  }
}`
	if err := os.WriteFile(cfg, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := NewModelMapper(cfg)
	if err == nil {
		t.Fatal("expected error for unknown capability, got nil")
	}
	if expected := "unknown capability"; !contains(err.Error(), expected) {
		t.Errorf("error %q should contain %q", err.Error(), expected)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
