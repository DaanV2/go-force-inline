package cmd

import (
	"os"

	"github.com/daanv2/go-force-inline/pkg/verifier"
	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	var threshold float64

	cmd := &cobra.Command{
		Use:   "verify <profile.pgo>",
		Short: "Verify which edges in a profile are hot",
		Long: `Reads a pprof profile, sorts edges by weight, prints each edge
with its CDF percentage, and marks which ones fall within the hot threshold.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return verifier.Verify(args[0], threshold, os.Stdout)
		},
	}

	cmd.Flags().Float64Var(&threshold, "threshold", 99.0, "CDF hot threshold percentage")

	return cmd
}
