package cmd

import (
	"os"

	"github.com/daanv2/go-force-inline/pkg/verifier"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <profile.pgo>",
	Short: "Verify which edges in a profile are hot",
	Long: `Reads a pprof profile, sorts edges by weight, prints each edge
with its CDF percentage, and marks which ones fall within the hot threshold.`,
	Args: cobra.ExactArgs(1),
	RunE: runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	verifyCmd.Flags().Float64("threshold", 99.0, "CDF hot threshold percentage")
}

func runVerify(cmd *cobra.Command, args []string) error {
	threshold, _ := cmd.Flags().GetFloat64("threshold")

	return verifier.Verify(args[0], threshold, os.Stdout)
}
