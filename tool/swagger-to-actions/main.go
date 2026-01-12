package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	cfgFile        string
	inputFile      string
	outputFile     string
	serviceName    string
	serviceVersion string
	serviceRepo    string
	appendMode     bool
	verbose        bool
	timeout        int
)

var rootCmd = &cobra.Command{
	Use:   "swagger-to-actions",
	Short: "Convert Swagger/OpenAPI specs to serviceActions YAML format",
	Long: `A CLI tool that transforms Swagger/OpenAPI specifications into
a serviceActions YAML format for use with MC-IAM-Manager.

Supports:
  - Multiple frameworks via configuration file
  - Swagger 2.0 and OpenAPI 3.0+ specifications
  - Both JSON and YAML input formats
  - Local files and remote URLs

Examples:
  # Multi-framework mode (recommended)
  swagger-to-actions -c frameworks.yaml

  # Single framework mode
  swagger-to-actions -i ./swagger.yaml -o ./actions.yaml -s mc-iam-manager

  # Append to existing file
  swagger-to-actions -i ./new-swagger.yaml -o ./existing.yaml -s new-service --append`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "Path to frameworks configuration file")
	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input Swagger/OpenAPI file path or URL (single mode)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output YAML file path")
	rootCmd.Flags().StringVarP(&serviceName, "service", "s", "", "Service name for single mode")
	rootCmd.Flags().StringVarP(&serviceVersion, "version", "V", "", "Service version for single mode (optional)")
	rootCmd.Flags().StringVarP(&serviceRepo, "repository", "r", "", "Repository URL for single mode (optional)")
	rootCmd.Flags().BoolVarP(&appendMode, "append", "a", false, "Append to existing output file")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "HTTP timeout in seconds for URL fetching")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	printBanner()

	// Validate flags
	if err := validateFlags(); err != nil {
		printError("%v", err)
		return err
	}

	// Determine mode: config-based or single
	if cfgFile != "" {
		return runConfigMode()
	}
	return runSingleMode()
}

func validateFlags() error {
	// Config mode: -c is required
	// Single mode: -i, -o, -s are required
	if cfgFile == "" && inputFile == "" {
		return fmt.Errorf("either --config (-c) or --input (-i) is required")
	}

	if cfgFile != "" && inputFile != "" {
		return fmt.Errorf("cannot use both --config and --input at the same time")
	}

	if inputFile != "" {
		if outputFile == "" {
			return fmt.Errorf("--output (-o) is required in single mode")
		}
		if serviceName == "" {
			return fmt.Errorf("--service (-s) is required in single mode")
		}
	}

	return nil
}

func runConfigMode() error {
	printInfo("Running in config mode with: %s", cfgFile)

	// Load configuration
	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		printError("Failed to load config: %v", err)
		return err
	}

	if err := cfg.Validate(); err != nil {
		printError("Invalid config: %v", err)
		return err
	}

	// Override verbose and timeout from flags if set
	if verbose {
		cfg.Verbose = true
	}
	if timeout != 30 {
		cfg.Timeout = timeout
	}

	if cfg.Verbose {
		printInfo("Output file: %s", cfg.Output)
		printInfo("Frameworks: %d", len(cfg.Frameworks))
		for _, fw := range cfg.Frameworks {
			printInfo("  - %s: %s", fw.Name, fw.Swagger)
		}
	}

	// Create aggregator and process
	agg := NewAggregator(cfg.Timeout, cfg.Verbose)
	output, err := agg.Process(cfg)
	if err != nil {
		printError("Failed to process frameworks: %v", err)
		return err
	}

	// Write output
	outputPath := cfg.Output
	if outputFile != "" {
		outputPath = outputFile
	}

	if err := WriteYAML(output, outputPath); err != nil {
		printError("Failed to write output: %v", err)
		return err
	}

	printSuccess("Successfully generated: %s", outputPath)
	printInfo("Total frameworks: %d", len(output.ServiceActions))

	totalActions := 0
	for name, actions := range output.ServiceActions {
		// Action count excludes _meta
		actionCount := len(actions) - 1
		printInfo("  - %s: %d actions", name, actionCount)
		totalActions += actionCount
	}
	printInfo("Total actions: %d", totalActions)

	return nil
}

func runSingleMode() error {
	printInfo("Running in single mode")
	printInfo("Input: %s", inputFile)
	printInfo("Service: %s", serviceName)
	if serviceVersion != "" {
		printInfo("Version: %s", serviceVersion)
	}

	// Create aggregator
	agg := NewAggregator(timeout, verbose)

	// Process single framework
	output, err := agg.ProcessSingle(inputFile, serviceName, serviceVersion, serviceRepo)
	if err != nil {
		printError("Failed to process: %v", err)
		return err
	}

	// Write or append output
	if appendMode {
		if err := AppendToYAML(output, outputFile); err != nil {
			printError("Failed to append to output: %v", err)
			return err
		}
		printSuccess("Successfully appended to: %s", outputFile)
	} else {
		if err := WriteYAML(output, outputFile); err != nil {
			printError("Failed to write output: %v", err)
			return err
		}
		printSuccess("Successfully generated: %s", outputFile)
	}

	// Action count excludes _meta
	actionCount := len(output.ServiceActions[serviceName]) - 1
	printInfo("Actions: %d", actionCount)

	return nil
}

func printBanner() {
	banner := `
 ____                                         _                  _   _
/ ___|_      ____ _  __ _  __ _  ___ _ __    | |_ ___           / \ | | ___ | |_ ___
\___ \ \ /\ / / _  |/ _  |/ _  |/ _ \ '__|___| __/ _ \  _____  / _ \| |/ __|| __/ _ \
 ___) \ V  V / (_| | (_| | (_| |  __/ | |____| || (_) ||_____|/ ___ \ | (__ | || (_) |
|____/ \_/\_/ \__,_|\__, |\__, |\___|_|       \__\___/       /_/   \_\_\___| \__\___/
                    |___/ |___/
`
	color.Cyan(banner)
	fmt.Println()
}

// Output helpers
func printInfo(format string, args ...interface{}) {
	color.Cyan("[INFO] "+format+"\n", args...)
}

func printSuccess(format string, args ...interface{}) {
	color.Green("[SUCCESS] "+format+"\n", args...)
}

func printWarning(format string, args ...interface{}) {
	color.Yellow("[WARN] "+format+"\n", args...)
}

func printError(format string, args ...interface{}) {
	color.Red("[ERROR] "+format+"\n", args...)
}
