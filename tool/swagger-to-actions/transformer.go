package main

// FrameworkMeta contains metadata for a framework
type FrameworkMeta struct {
	Version     string `yaml:"version"`
	Repository  string `yaml:"repository"`
	GeneratedAt string `yaml:"generatedAt"`
}

// ServiceAction represents a single service action
type ServiceAction struct {
	Method       string `yaml:"method"`
	ResourcePath string `yaml:"resourcePath"`
	Description  string `yaml:"description"`
}

// Transform converts a SwaggerSpec to a map of ServiceActions
func Transform(spec *SwaggerSpec) (map[string]ServiceAction, error) {
	actions := make(map[string]ServiceAction)

	operations := spec.GetOperations()

	for _, op := range operations {
		// Use operationId as the action name
		actionName := op.Operation.OperationID
		if actionName == "" {
			continue // Skip operations without operationId
		}

		// Get description, fallback to summary
		description := op.Operation.Description
		if description == "" {
			description = op.Operation.Summary
		}

		actions[actionName] = ServiceAction{
			Method:       op.Method,
			ResourcePath: op.Path,
			Description:  description,
		}
	}

	return actions, nil
}
