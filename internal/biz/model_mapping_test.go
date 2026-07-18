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
	content := `models:
  gpt-4o:
    actual_name: gpt-4o-2024-08-06
    capabilities:
      - streaming
  claude-3-5-sonnet:
    actual_name: claude-3-5-sonnet-20241022
    capabilities:
      - vision
      - streaming
`
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
	content := `models:
  gpt-4o:
    actual_name: gpt-4o-2024-08-06
    capabilities:
      - function_call
      - vision
      - streaming
`
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
	content := `models:
  gpt-4o:
    actual_name: ""
    capabilities:
      - streaming
`
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
	content := `models:
  gpt-4o:
    actual_name: gpt-4o-2024-08-06
    capabilities:
      - streaming
      - telepathy
`
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

func TestModelMapper_YAMLFormat(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "models.yaml")
	content := `# Model config in YAML format
models:
  gpt-4o:
    actual_name: gpt-4o-2024-08-06
    capabilities:
      - function_call
      - vision
      - streaming
  claude-3-5-sonnet:
    actual_name: claude-3-5-sonnet-20241022
    capabilities:
      - vision
      - streaming
`
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
	if got := m.Resolve("claude-3-5-sonnet"); got != "claude-3-5-sonnet-20241022" {
		t.Errorf("Resolve(claude-3-5-sonnet) = %s, want claude-3-5-sonnet-20241022", got)
	}
	if !m.HasCapability("gpt-4o", "vision") {
		t.Error("expected gpt-4o to have vision capability")
	}
}

func TestModelMapper_JSONFormat(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "models.json")
	content := `{
  "models": {
    "gpt-4o": {"actual_name": "gpt-4o-2024-08-06", "capabilities": ["streaming"]}
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

// ---------------------------------------------------------------------------
// Phase 2.5 — hot reload contract.
// ---------------------------------------------------------------------------

func TestModelMapper_Reload_PicksUpNewEntries(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/models.yaml"
	if err := os.WriteFile(path, []byte("models:\n  gpt-4o:\n    actual_name: gpt-4o-2024-08-06\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := NewModelMapper(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Resolve("gpt-4o"); got != "gpt-4o-2024-08-06" {
		t.Fatalf("initial resolve: got %q, want gpt-4o-2024-08-06", got)
	}
	if got := m.Resolve("claude-3"); got != "claude-3" {
		t.Fatalf("unknown model should pass through, got %q", got)
	}

	// Rewrite file to add claude-3 + change gpt-4o target.
	if err := os.WriteFile(path, []byte("models:\n  gpt-4o:\n    actual_name: gpt-4o-2025-01-01\n  claude-3:\n    actual_name: claude-3-opus\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := m.Resolve("gpt-4o"); got != "gpt-4o-2025-01-01" {
		t.Fatalf("after reload: gpt-4o got %q, want gpt-4o-2025-01-01", got)
	}
	if got := m.Resolve("claude-3"); got != "claude-3-opus" {
		t.Fatalf("after reload: claude-3 got %q, want claude-3-opus", got)
	}
}

func TestModelMapper_Reload_RejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/models.yaml"
	if err := os.WriteFile(path, []byte("models:\n  good:\n    actual_name: upstream\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := NewModelMapper(path)
	if err != nil {
		t.Fatal(err)
	}
	// Now write an invalid file (empty actual_name). Reload must fail and the
	// previous snapshot must remain observable.
	if err := os.WriteFile(path, []byte("models:\n  bad:\n    actual_name: ''\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.Reload(); err == nil {
		t.Fatal("Reload should reject empty actual_name")
	}
	if got := m.Resolve("good"); got != "upstream" {
		t.Fatalf("after failed reload, snapshot should be unchanged: got %q, want upstream", got)
	}
}
