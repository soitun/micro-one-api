package biz

import (
	"fmt"
	"os"
	"strings"

	"github.com/bytedance/sonic"
	"gopkg.in/yaml.v3"
)

// ModelEntry represents a single model mapping entry.
type ModelEntry struct {
	ActualName   string   `json:"actual_name" yaml:"actual_name"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
}

// modelsFile is the YAML/JSON structure of models config.
type modelsFile struct {
	Models map[string]*ModelEntry `json:"models" yaml:"models"`
}

// KnownCapabilities is the set of valid model capabilities.
var KnownCapabilities = map[string]bool{
	"function_call": true,
	"vision":        true,
	"streaming":     true,
	"embedding":     true,
}

// ModelMapper resolves client-facing model names to actual upstream model names.
type ModelMapper struct {
	models map[string]*ModelEntry
}

// NewModelMapper creates a ModelMapper from a config file path.
// Returns a no-op mapper if path is empty.
// Validates that all entries have non-empty actual_name and known capabilities.
func NewModelMapper(path string) (*ModelMapper, error) {
	if path == "" {
		return &ModelMapper{models: make(map[string]*ModelEntry)}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read models config %s: %w", path, err)
	}

	var file modelsFile
	if isYAMLFile(path) {
		if err := yaml.Unmarshal(data, &file); err != nil {
			return nil, fmt.Errorf("failed to parse models config %s: %w", path, err)
		}
	} else {
		if err := sonic.Unmarshal(data, &file); err != nil {
			return nil, fmt.Errorf("failed to parse models config %s: %w", path, err)
		}
	}

	if file.Models == nil {
		file.Models = make(map[string]*ModelEntry)
	}

	// Validate entries
	for name, entry := range file.Models {
		if entry.ActualName == "" {
			return nil, fmt.Errorf("models config: model %q has empty actual_name", name)
		}
		for _, cap := range entry.Capabilities {
			if !KnownCapabilities[cap] {
				return nil, fmt.Errorf("models config: model %q has unknown capability %q (known: function_call, vision, streaming, embedding)", name, cap)
			}
		}
	}

	return &ModelMapper{models: file.Models}, nil
}

// Resolve returns the actual upstream model name for the given client model name.
// If no mapping exists, returns the original name unchanged.
func (m *ModelMapper) Resolve(modelName string) string {
	if entry, ok := m.models[modelName]; ok && entry.ActualName != "" {
		return entry.ActualName
	}
	return modelName
}

// HasCapability checks if a model has the specified capability.
func (m *ModelMapper) HasCapability(modelName, capability string) bool {
	entry, ok := m.models[modelName]
	if !ok {
		return false
	}
	for _, cap := range entry.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// GetEntry returns the full model entry for the given name, or nil if not found.
func (m *ModelMapper) GetEntry(modelName string) *ModelEntry {
	return m.models[modelName]
}

// isYAMLFile checks if a file path has a YAML extension.
func isYAMLFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}
