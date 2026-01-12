package main

import (
	"fmt"
	"time"
)

// ServiceActionsOutput represents the final output structure
type ServiceActionsOutput struct {
	ServiceActions map[string]map[string]interface{} `yaml:"serviceActions"`
}

// Aggregator handles processing of multiple frameworks
type Aggregator struct {
	timeout int
	verbose bool
}

// NewAggregator creates a new Aggregator
func NewAggregator(timeout int, verbose bool) *Aggregator {
	return &Aggregator{
		timeout: timeout,
		verbose: verbose,
	}
}

// Process processes all frameworks from the config and aggregates results
func (a *Aggregator) Process(cfg *Config) (*ServiceActionsOutput, error) {
	output := &ServiceActionsOutput{
		ServiceActions: make(map[string]map[string]interface{}),
	}

	successCount := 0
	failCount := 0
	generatedAt := time.Now().UTC().Format(time.RFC3339)

	// Process each framework from config (no hardcoding!)
	for _, fw := range cfg.Frameworks {
		if a.verbose {
			printInfo("Processing framework: %s", fw.Name)
			printInfo("  Swagger: %s", fw.Swagger)
		}

		actions, err := a.processFramework(fw)
		if err != nil {
			printWarning("Failed to process %s: %v", fw.Name, err)
			failCount++
			continue
		}

		// Create framework output with _meta and actions
		frameworkOutput := make(map[string]interface{})

		// Add metadata
		frameworkOutput["_meta"] = FrameworkMeta{
			Version:     fw.Version,
			Repository:  fw.Repository,
			GeneratedAt: generatedAt,
		}

		// Add all actions
		for name, action := range actions {
			frameworkOutput[name] = action
		}

		output.ServiceActions[fw.Name] = frameworkOutput
		successCount++

		if a.verbose {
			printInfo("  Actions: %d", len(actions))
		}
	}

	if successCount == 0 {
		return nil, fmt.Errorf("all frameworks failed to process")
	}

	if failCount > 0 {
		printWarning("Completed with %d failures out of %d frameworks", failCount, len(cfg.Frameworks))
	}

	return output, nil
}

// ProcessSingle processes a single swagger file
func (a *Aggregator) ProcessSingle(input, serviceName, version, repository string) (*ServiceActionsOutput, error) {
	fw := Framework{
		Name:       serviceName,
		Version:    version,
		Repository: repository,
		Swagger:    input,
	}

	actions, err := a.processFramework(fw)
	if err != nil {
		return nil, err
	}

	generatedAt := time.Now().UTC().Format(time.RFC3339)

	// Create framework output with _meta and actions
	frameworkOutput := make(map[string]interface{})

	// Add metadata
	frameworkOutput["_meta"] = FrameworkMeta{
		Version:     fw.Version,
		Repository:  fw.Repository,
		GeneratedAt: generatedAt,
	}

	// Add all actions
	for name, action := range actions {
		frameworkOutput[name] = action
	}

	output := &ServiceActionsOutput{
		ServiceActions: map[string]map[string]interface{}{
			serviceName: frameworkOutput,
		},
	}

	return output, nil
}

// processFramework processes a single framework
func (a *Aggregator) processFramework(fw Framework) (map[string]ServiceAction, error) {
	var data []byte
	var err error

	// Fetch or read the swagger file
	if IsURL(fw.Swagger) {
		if a.verbose {
			printInfo("  Fetching from URL...")
		}
		data, err = FetchURL(fw.Swagger, a.timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch swagger: %w", err)
		}
	} else {
		if a.verbose {
			printInfo("  Reading from file...")
		}
		spec, err := ParseFile(fw.Swagger)
		if err != nil {
			return nil, fmt.Errorf("failed to parse swagger file: %w", err)
		}

		if a.verbose {
			printInfo("  Version: %s", spec.GetVersion())
		}

		return Transform(spec)
	}

	// Parse the fetched data
	spec, err := ParseBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse swagger: %w", err)
	}

	if a.verbose {
		printInfo("  Version: %s", spec.GetVersion())
	}

	// Transform to service actions
	return Transform(spec)
}
