package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/daanv2/go-force-inline/pkg/profgen"
	"github.com/daanv2/go-force-inline/pkg/resolver"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate [packages]",
	Short: "Generate a synthetic PGO profile from //pgogen:hot directives",
	Long: `Scans Go source files matching the given package patterns for
//pgogen:hot directives, resolves caller/callee linker symbols,
and writes a synthetic pprof profile.`,
	Example: "go-force-inline generate ./...",
	Aliases: []string{"gen"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringP("output", "o", "default.pgo", "Output file path")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")

	log.Info("scanning packages", "patterns", args)

	edges, err := resolver.Resolve(args)
	if err != nil {
		return err
	}

	if len(edges) == 0 {
		log.Warn("no //pgogen:hot directives found")

		return nil
	}

	log.Info("found directives", "count", len(edges))

	return profgen.Generate(edges, output)
}
