package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "pgogen",
		Short: "Generate synthetic PGO profiles from source directives",
		Long: `pgogen scans Go source files for //pgogen:hot comment directives and
generates a synthetic pprof CPU profile (default.pgo) that causes the
Go compiler's PGO inliner to treat annotated call sites as "hot."`,
	}

	root.AddCommand(
		newGenerateCmd(),
		newVerifyCmd(),
	)

	return root
}
