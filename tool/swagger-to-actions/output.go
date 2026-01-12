package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// WriteYAML writes the ServiceActionsOutput to a YAML file
func WriteYAML(output *ServiceActionsOutput, path string) error {
	data, err := yaml.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// AppendToYAML appends new service actions to an existing YAML file
func AppendToYAML(output *ServiceActionsOutput, path string) error {
	// Read existing file
	existingData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, just write new content
			return WriteYAML(output, path)
		}
		return fmt.Errorf("failed to read existing file: %w", err)
	}

	// Parse existing content
	existing := &ServiceActionsOutput{}
	if err := yaml.Unmarshal(existingData, existing); err != nil {
		return fmt.Errorf("failed to parse existing YAML: %w", err)
	}

	// Merge new actions into existing
	if existing.ServiceActions == nil {
		existing.ServiceActions = make(map[string]map[string]interface{})
	}

	for serviceName, actions := range output.ServiceActions {
		if existing.ServiceActions[serviceName] == nil {
			existing.ServiceActions[serviceName] = make(map[string]interface{})
		}
		for actionName, action := range actions {
			existing.ServiceActions[serviceName][actionName] = action
		}
	}

	// Write merged content
	return WriteYAML(existing, path)
}
