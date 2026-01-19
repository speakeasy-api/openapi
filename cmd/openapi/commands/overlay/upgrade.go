package overlay

import (
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade <overlay-file> [output-file]",
	Short: "Upgrade an Overlay document to the latest supported version (1.1.0)",
	Long: `Upgrade an Overlay specification document to the latest supported version (1.1.0).

The upgrade process includes:
- Updating the Overlay version field from 1.0.0 to 1.1.0
- Enabling RFC 9535 JSONPath as the default implementation
- Clearing redundant x-speakeasy-jsonpath: rfc9535 (now default in 1.1.0)
- All existing actions remain valid and functional
- Support for new 1.1.0 features like copy actions and info description

Version Differences:
  1.0.0: Legacy JSONPath by default, RFC 9535 opt-in with x-speakeasy-jsonpath: rfc9535
  1.1.0: RFC 9535 JSONPath by default, legacy opt-out with x-speakeasy-jsonpath: legacy

Output options:
  - No output file specified: writes to stdout (pipe-friendly)
  - Output file specified: writes to the specified file
  - --write flag: writes in-place to the input file`,
	Example: `  # Preview upgrade (output to stdout)
  openapi overlay upgrade my-overlay.yaml

  # Upgrade and save to new file
  openapi overlay upgrade my-overlay.yaml upgraded-overlay.yaml

  # Upgrade in-place
  openapi overlay upgrade -w my-overlay.yaml`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runOverlayUpgrade,
}

var overlayWriteInPlace bool

func init() {
	upgradeCmd.Flags().BoolVarP(&overlayWriteInPlace, "write", "w", false,
		"write result in-place to input file")
}

func runOverlayUpgrade(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := args[0]

	var outputFile string
	if len(args) > 1 {
		outputFile = args[1]
	}

	// Load the overlay
	o, err := loader.LoadOverlay(inputFile)
	if err != nil {
		Dief("Failed to load overlay: %v", err)
	}

	// Validate the overlay before upgrade
	if err := o.Validate(); err != nil {
		Dief("Overlay validation failed: %v", err)
	}

	originalVersion := o.Version

	// Perform the upgrade
	upgraded, err := overlay.Upgrade(ctx, o)
	if err != nil {
		Dief("Failed to upgrade overlay: %v", err)
	}

	// Print status
	if !upgraded {
		fmt.Fprintf(os.Stderr, "No upgrade needed - overlay is already at version %s\n", originalVersion)
	} else {
		fmt.Fprintf(os.Stderr, "Successfully upgraded overlay from %s to %s\n", originalVersion, o.Version)
	}

	// Validate the upgraded overlay
	if err := o.Validate(); err != nil {
		Dief("Upgraded overlay failed validation: %v", err)
	}

	// Serialize output
	output, err := o.ToString()
	if err != nil {
		Dief("Failed to serialize overlay: %v", err)
	}

	// Determine output destination
	switch {
	case overlayWriteInPlace:
		if err := os.WriteFile(inputFile, []byte(output), 0644); err != nil {
			Dief("Failed to write to input file: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Wrote upgraded overlay to %s\n", inputFile)
	case outputFile != "":
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			Dief("Failed to write to output file: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Wrote upgraded overlay to %s\n", outputFile)
	default:
		// Write to stdout
		var node yaml.Node
		if err := yaml.Unmarshal([]byte(output), &node); err != nil {
			Dief("Failed to parse output: %v", err)
		}
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		if err := encoder.Encode(&node); err != nil {
			Dief("Failed to write to stdout: %v", err)
		}
	}
}
