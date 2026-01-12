package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SwaggerSpec represents a Swagger 2.0 or OpenAPI 3.0 specification
type SwaggerSpec struct {
	Swagger string              `json:"swagger" yaml:"swagger"` // Swagger 2.0
	OpenAPI string              `json:"openapi" yaml:"openapi"` // OpenAPI 3.0+
	Info    SwaggerInfo         `json:"info" yaml:"info"`
	Paths   map[string]PathItem `json:"paths" yaml:"paths"`
}

// SwaggerInfo contains API metadata
type SwaggerInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

// PathItem represents operations available on a single path
type PathItem struct {
	Get     *Operation `json:"get" yaml:"get"`
	Post    *Operation `json:"post" yaml:"post"`
	Put     *Operation `json:"put" yaml:"put"`
	Delete  *Operation `json:"delete" yaml:"delete"`
	Patch   *Operation `json:"patch" yaml:"patch"`
	Options *Operation `json:"options" yaml:"options"`
	Head    *Operation `json:"head" yaml:"head"`
}

// Operation represents a single API operation
type Operation struct {
	OperationID string   `json:"operationId" yaml:"operationId"`
	Summary     string   `json:"summary" yaml:"summary"`
	Description string   `json:"description" yaml:"description"`
	Tags        []string `json:"tags" yaml:"tags"`
}

// ParseFile parses a Swagger/OpenAPI file from a path
func ParseFile(path string) (*SwaggerSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseBytes(data)
}

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

// GetOperations returns all operations from the spec with their paths and methods
func (s *SwaggerSpec) GetOperations() []OperationInfo {
	var operations []OperationInfo

	for path, pathItem := range s.Paths {
		if pathItem.Get != nil && pathItem.Get.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "get",
				Operation: pathItem.Get,
			})
		}
		if pathItem.Post != nil && pathItem.Post.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "post",
				Operation: pathItem.Post,
			})
		}
		if pathItem.Put != nil && pathItem.Put.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "put",
				Operation: pathItem.Put,
			})
		}
		if pathItem.Delete != nil && pathItem.Delete.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "delete",
				Operation: pathItem.Delete,
			})
		}
		if pathItem.Patch != nil && pathItem.Patch.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "patch",
				Operation: pathItem.Patch,
			})
		}
		if pathItem.Options != nil && pathItem.Options.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "options",
				Operation: pathItem.Options,
			})
		}
		if pathItem.Head != nil && pathItem.Head.OperationID != "" {
			operations = append(operations, OperationInfo{
				Path:      path,
				Method:    "head",
				Operation: pathItem.Head,
			})
		}
	}

	return operations
}

// OperationInfo holds operation details with path and method
type OperationInfo struct {
	Path      string
	Method    string
	Operation *Operation
}
