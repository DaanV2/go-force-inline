package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/daanv2/go-force-inline/pkg/profgen"
	"github.com/daanv2/go-force-inline/pkg/resolver"
	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	var (
		output  string
		verbose bool
	)

	cmd := &cobra.Command{
		Use:   "generate [packages]",
		Short: "Generate a synthetic PGO profile from //pgogen:hot directives",
		Long: `Scans Go source files matching the given package patterns for
//pgogen:hot directives, resolves caller/callee linker symbols,
and writes a synthetic pprof profile.`,
		Aliases: []string{"gen"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if verbose {
				log.SetLevel(log.DebugLevel)
			}

			log.Info("scanning packages", "patterns", args)

			edges, err := resolver.Resolve(args, verbose)
			if err != nil {
				return err
			}

			if len(edges) == 0 {
				log.Warn("no //pgogen:hot directives found")
				return nil
			}

			log.Info("found directives", "count", len(edges))

			return profgen.Generate(edges, output, verbose)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "default.pgo", "Output file path")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output: print discovered directives and generated edges")

	return cmd
}
