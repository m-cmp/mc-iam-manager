package apiparser

import (
	"fmt"
)

// Processor handles API specification processing
type Processor struct {
	TimeoutSeconds int
}

// NewProcessor creates a new Processor with the specified timeout
func NewProcessor(timeoutSeconds int) *Processor {
	return &Processor{
		TimeoutSeconds: timeoutSeconds,
	}
}

// ProcessFramework fetches, parses, and transforms a framework's API specification
func (p *Processor) ProcessFramework(name, version, repository, sourceType, sourceURL string) *FrameworkResult {
	result := &FrameworkResult{
		Name:       name,
		Version:    version,
		Repository: repository,
	}

	// Validate source type
	if sourceType != string(SourceTypeSwagger) && sourceType != string(SourceTypeOpenAPI) {
		result.Error = fmt.Errorf("unsupported source type: %s (supported: swagger, openapi)", sourceType)
		return result
	}

	// Fetch the specification
	data, err := FetchURL(sourceURL, p.TimeoutSeconds)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch: %w", err)
		return result
	}

	// Parse the specification
	spec, err := ParseBytes(data)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse: %w", err)
		return result
	}

	// Transform to actions
	actions, err := Transform(spec)
	if err != nil {
		result.Error = fmt.Errorf("failed to transform: %w", err)
		return result
	}

	result.Actions = actions
	result.ActionCount = len(actions)

	return result
}

// ProcessFrameworkFromBytes parses and transforms from raw bytes (for testing or local files)
func (p *Processor) ProcessFrameworkFromBytes(name, version, repository string, data []byte) *FrameworkResult {
	result := &FrameworkResult{
		Name:       name,
		Version:    version,
		Repository: repository,
	}

	// Parse the specification
	spec, err := ParseBytes(data)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse: %w", err)
		return result
	}

	// Transform to actions
	actions, err := Transform(spec)
	if err != nil {
		result.Error = fmt.Errorf("failed to transform: %w", err)
		return result
	}

	result.Actions = actions
	result.ActionCount = len(actions)

	return result
}
