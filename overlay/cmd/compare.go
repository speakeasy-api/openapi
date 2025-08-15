package cmd

import (
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/speakeasy-api/openapi/overlay/loader"
	"github.com/spf13/cobra"
)

var (
	compareCmd = &cobra.Command{
		Use:   "compare <spec1> <spec2>",
		Short: "Given two specs, it will output an overlay that describes the differences between them",
		Args:  cobra.ExactArgs(2),
		Run:   RunCompare,
	}
)

func RunCompare(cmd *cobra.Command, args []string) {
	y1, err := loader.LoadSpecification(args[0])
	if err != nil {
		Dief("Failed to load %q: %v", args[0], err)
	}

	y2, err := loader.LoadSpecification(args[1])
	if err != nil {
		Dief("Failed to load %q: %v", args[1], err)
	}

	title := fmt.Sprintf("Overlay %s => %s", args[0], args[1])

	o, err := overlay.Compare(title, y1, *y2)
	if err != nil {
		Dief("Failed to compare spec files %q and %q: %v", args[0], args[1], err)
	}

	err = o.Format(os.Stdout)
	if err != nil {
		Dief("Failed to format overlay: %v", err)
	}
}
