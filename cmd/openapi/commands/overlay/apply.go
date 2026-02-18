package overlay

import (
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	overlayPkg "github.com/speakeasy-api/openapi/overlay"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
)

var (
	applyOverlayFlags []string
	applySchemaFlag   string
	applyOutFlag      string
)

var applyCmd = &cobra.Command{
	Use:   "apply [<overlay> [<spec>]]",
	Short: "Given one or more overlays, it will apply them sequentially to the spec. If omitted, spec will be loaded via extends (only from local file system).",
	Args:  cobra.RangeArgs(0, 2),
	Run:   RunApply,
	Example: `  # Apply overlay using positional arguments
  openapi overlay apply overlay.yaml spec.yaml

  # Apply overlay using flags
  openapi overlay apply --overlay overlay.yaml --schema spec.yaml

  # Apply overlay with output to file
  openapi overlay apply --overlay overlay.yaml --schema spec.yaml --out modified-spec.yaml

  # Apply overlay when overlay has extends key set
  openapi overlay apply overlay.yaml

  # Pipe spec via stdin and provide overlay as file
  cat spec.yaml | openapi overlay apply overlay.yaml -
  cat spec.yaml | openapi overlay apply --overlay overlay.yaml --schema -

  # Apply multiple overlays sequentially (repeated flag)
  openapi overlay apply --overlay base.yaml --overlay env.yaml --schema spec.yaml`,
}

func init() {
	applyCmd.Flags().StringSliceVar(&applyOverlayFlags, "overlay", nil, "Path to an overlay file (can be repeated or comma-separated)")
	applyCmd.Flags().StringVar(&applySchemaFlag, "schema", "", "Path to the OpenAPI specification file (use '-' for stdin)")
	applyCmd.Flags().StringVarP(&applyOutFlag, "out", "o", "", "Output file path (defaults to stdout)")
}

func RunApply(cmd *cobra.Command, args []string) {
	// Build the list of overlay files from flags or positional args
	var overlayFiles []string
	if len(applyOverlayFlags) > 0 {
		overlayFiles = applyOverlayFlags
	} else if len(args) > 0 {
		overlayFiles = []string{args[0]}
	} else {
		Dief("at least one overlay file is required (use --overlay flag or provide as first argument)")
	}

	// Load all overlays upfront (fail fast on parse errors)
	overlays := make([]*overlayPkg.Overlay, 0, len(overlayFiles))
	for _, f := range overlayFiles {
		o, err := loader.LoadOverlay(f)
		if err != nil {
			Dief("Failed to load overlay %q: %s", f, err.Error())
		}
		overlays = append(overlays, o)
	}

	// Determine spec file path from flag or positional argument
	var specFile string
	if applySchemaFlag != "" {
		specFile = applySchemaFlag
	} else {
		specFile = cmdutil.ArgAt(args, 1, "")
	}

	// Load spec from stdin or file (use first overlay's extends as fallback)
	var ys *yaml.Node
	var err error
	specSource := specFile
	if cmdutil.IsStdin(specFile) || (specFile == "" && cmdutil.StdinIsPiped()) {
		fmt.Fprintf(os.Stderr, "Reading specification from stdin\n")
		specSource = "stdin"
		ys, err = loader.LoadSpecificationFromReader(os.Stdin)
		if err != nil {
			Die(err)
		}
	} else {
		ys, specSource, err = loader.LoadEitherSpecification(specFile, overlays[0])
		if err != nil {
			Die(err)
		}
	}

	// Apply all overlays sequentially
	for i, o := range overlays {
		if err := o.ApplyTo(ys); err != nil {
			Dief("Failed to apply overlay %q (#%d) to spec %q: %v", overlayFiles[i], i+1, specSource, err)
		}
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
		Dief("Failed to encode spec %q: %v", specSource, err)
	}
}
