package cmd

import (
	"os"

	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	applyCmd = &cobra.Command{
		Use:   "apply <overlay> [ <spec> ]",
		Short: "Given an overlay, it will apply it to the spec. If omitted, spec will be loaded via extends (only from local file system).",
		Args:  cobra.RangeArgs(1, 2),
		Run:   RunApply,
	}
)

func RunApply(cmd *cobra.Command, args []string) {
	overlayFile := args[0]

	o, err := loader.LoadOverlay(overlayFile)
	if err != nil {
		Die(err)
	}

	var specFile string
	if len(args) > 1 {
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

	err = yaml.NewEncoder(os.Stdout).Encode(ys)
	if err != nil {
		Dief("Failed to encode spec file %q: %v", specFile, err)
	}
}
