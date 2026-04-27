package apiparser

// SourceType represents the type of API specification source
type SourceType string

const (
	SourceTypeSwagger SourceType = "swagger" // Swagger 2.0
	SourceTypeOpenAPI SourceType = "openapi" // OpenAPI 3.0+
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

// OperationInfo holds operation details with path and method
type OperationInfo struct {
	Path      string
	Method    string
	Operation *Operation
}

// FrameworkMeta contains metadata for a framework
type FrameworkMeta struct {
	Version     string `yaml:"version" json:"version"`
	Repository  string `yaml:"repository" json:"repository"`
	GeneratedAt string `yaml:"generatedAt" json:"generatedAt"`
}

// ServiceAction represents a single service action
type ServiceAction struct {
	Method       string `yaml:"method" json:"method"`
	ResourcePath string `yaml:"resourcePath" json:"resourcePath"`
	Description  string `yaml:"description" json:"description"`
}

// FrameworkResult holds the result of processing a single framework
type FrameworkResult struct {
	Name        string
	Version     string
	Repository  string
	Actions     map[string]ServiceAction
	ActionCount int
	Error       error
}
