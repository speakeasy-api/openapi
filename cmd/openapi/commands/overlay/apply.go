package overlay

import (
	"os"

	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	applyOverlayFlag string
	applySchemaFlag  string
	applyOutFlag     string
)

var applyCmd = &cobra.Command{
	Use:   "apply <overlay> [ <spec> ]",
	Short: "Given an overlay, it will apply it to the spec. If omitted, spec will be loaded via extends (only from local file system).",
	Args:  cobra.RangeArgs(0, 2),
	Run:   RunApply,
	Example: `  # Apply overlay using positional arguments
  openapi overlay apply overlay.yaml spec.yaml

  # Apply overlay using flags
  openapi overlay apply --overlay overlay.yaml --schema spec.yaml

  # Apply overlay with output to file
  openapi overlay apply --overlay overlay.yaml --schema spec.yaml --out modified-spec.yaml

  # Apply overlay when overlay has extends key set
  openapi overlay apply overlay.yaml`,
}

func init() {
	applyCmd.Flags().StringVar(&applyOverlayFlag, "overlay", "", "Path to the overlay file")
	applyCmd.Flags().StringVar(&applySchemaFlag, "schema", "", "Path to the OpenAPI specification file")
	applyCmd.Flags().StringVarP(&applyOutFlag, "out", "o", "", "Output file path (defaults to stdout)")
}

func RunApply(cmd *cobra.Command, args []string) {
	// Determine overlay file path from flag or positional argument
	var overlayFile string
	if applyOverlayFlag != "" {
		overlayFile = applyOverlayFlag
	} else if len(args) > 0 {
		overlayFile = args[0]
	} else {
		Dief("overlay file is required (use --overlay flag or provide as first argument)")
	}

	o, err := loader.LoadOverlay(overlayFile)
	if err != nil {
		Die(err)
	}

	// Determine spec file path from flag or positional argument
	var specFile string
	if applySchemaFlag != "" {
		specFile = applySchemaFlag
	} else if len(args) > 1 {
		specFile = args[1]
	}

	ys, specFile, err := loader.LoadEitherSpecification(specFile, o)
	if err != nil {
		Die(err)
	}

	err = o.ApplyTo(ys)
	if err != nil {
		Dief("Failed to apply overlay to spec file %q: %v", specFile, err)
	}

	// Write to output file if specified, otherwise stdout
	out := os.Stdout
	if applyOutFlag != "" {
		f, err := os.Create(applyOutFlag)
		if err != nil {
			Dief("Failed to create output file %q: %v", applyOutFlag, err)
		}
		defer f.Close()
		out = f
	}

	err = yaml.NewEncoder(out).Encode(ys)
	if err != nil {
		Dief("Failed to encode spec file %q: %v", specFile, err)
	}
}
