package apiparser

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseBytes parses Swagger/OpenAPI data from bytes
func ParseBytes(data []byte) (*SwaggerSpec, error) {
	format := DetectFormat(data)

	spec := &SwaggerSpec{}

	switch format {
	case "json":
		if err := json.Unmarshal(data, spec); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case "yaml":
		if err := yaml.Unmarshal(data, spec); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown format")
	}

	// Validate that it's a valid Swagger/OpenAPI spec
	if spec.Swagger == "" && spec.OpenAPI == "" {
		return nil, fmt.Errorf("not a valid Swagger/OpenAPI specification")
	}

	return spec, nil
}

// DetectFormat detects whether the data is JSON or YAML
func DetectFormat(data []byte) string {
	trimmed := strings.TrimSpace(string(data))

	// JSON typically starts with { or [
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "json"
	}

	// Default to YAML
	return "yaml"
}

// GetVersion returns the Swagger/OpenAPI version
func (s *SwaggerSpec) GetVersion() string {
	if s.Swagger != "" {
		return "Swagger " + s.Swagger
	}
	if s.OpenAPI != "" {
		return "OpenAPI " + s.OpenAPI
	}
	return "Unknown"
}

// GetSourceType returns the source type based on the spec
func (s *SwaggerSpec) GetSourceType() SourceType {
	if s.Swagger != "" {
		return SourceTypeSwagger
	}
	if s.OpenAPI != "" {
		return SourceTypeOpenAPI
	}
	return SourceTypeSwagger // default
}
