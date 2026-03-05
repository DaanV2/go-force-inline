package cmd

import (
	"context"
	"errors"
	"os"
	"syscall"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/log"
	"github.com/daanv2/go-force-inline/internal/logging"
	"github.com/spf13/cobra"
)

// rootCmd is the top-level cobra command.
var rootCmd = &cobra.Command{
	Use:   "go-force-inline",
	Short: "Generate synthetic PGO profiles from source directives",
	Long: `go-force-inline scans Go source files for //pgogen:hot comment directives and
generates a synthetic pprof CPU profile (default.pgo) that causes the
Go compiler's PGO inliner to treat annotated call sites as "hot."`,
	Example:          `go-force-inline generate ./...\ngo-force-inline verify default.pgo`,
	PersistentPreRun: logging.ApplyLoggerFlags,
}

func init() {
	logging.LoggerFlags(rootCmd.PersistentFlags())
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	ctx := context.Background()

	defer func() {
		if e := recover(); e != nil {
			log.Fatal("uncaught error", "error", e)
		}
	}()

	err := fang.Execute(
		ctx,
		rootCmd,
		fang.WithNotifySignal(syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT),
	)
	if err != nil && !errors.Is(err, context.Canceled) {
		// nolint:gocritic // exitAfterDefer fine in this case, we already report the error
		log.Fatal("error during executing command", "error", err)
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
