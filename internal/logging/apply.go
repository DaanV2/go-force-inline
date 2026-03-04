package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// LoggerFlags adds logger-related flags to the given flag set.
func LoggerFlags(pflags *pflag.FlagSet) {
	pflags.Bool("log-report-caller", false, "Whenever or not to output the file that outputs the log")
	pflags.String("log-level", "info", "The debug level, levels are: debug, info, warn, error, fatal")
	pflags.String("log-format", "text", "The text format of the logger")
	pflags.String("log-file", "", "The text format of the logger")
}

// ApplyLoggerFlags configures the logger based on command line flags.
func ApplyLoggerFlags(cmd *cobra.Command, args []string) {
	logOptions := log.Options{
		TimeFormat:      time.DateTime,
		ReportCaller:    cmd.Flag("log-report-caller").Value.String() == "true",
		ReportTimestamp: false,
	}

	// log-level
	level, err := log.ParseLevel(cmd.Flag("log-level").Value.String())
	if err != nil {
		log.Fatal("invalid log level", "error", err)
	}
	logOptions.Level = level

	// log-format
	switch cmd.Flag("log-format").Value.String() {
	default:
		logOptions.Formatter = log.TextFormatter
	case "json":
		logOptions.Formatter = log.JSONFormatter
	case "logfmt":
		logOptions.Formatter = log.LogfmtFormatter
	}

	var w io.Writer = os.Stderr
	fname := cmd.Flag("log-file").Value.String()
	if fname != "" {
		fname = filepath.Clean(fname)
		f, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			panic(fmt.Errorf("trouble with log file %w", err))
		}
		w = &splitWriter{
			base:  f,
			other: w,
		}

		runtime.SetFinalizer(w, func(*splitWriter) {
			_ = f.Close()
		})
	}

	// Initialize the default logger.
	logger := log.NewWithOptions(w, logOptions)
	logger.SetStyles(CreateStyle())
	log.SetDefault(logger)
}

// CreateStyle creates and returns custom log styles.
func CreateStyle() *log.Styles {
	styles := log.DefaultStyles()

	styles.Keys["err"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Keys["error"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["err"] = lipgloss.NewStyle().Bold(true)
	styles.Values["error"] = lipgloss.NewStyle().Bold(true)

	return styles
}

var _ io.Writer = &splitWriter{}

type splitWriter struct {
	base  io.Writer
	other io.Writer
}

// Write implements io.Writer.
func (s *splitWriter) Write(p []byte) (n int, err error) {
	n, err = s.base.Write(p)
	_, _ = s.other.Write(p)

	return
}
