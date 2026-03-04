package xio

import (
	"io"

	"github.com/charmbracelet/log"
)

// CloseReport closes the given io.Closer and reports any error to the provided logger.
// If the logger is nil, the default logger is used.
func CloseReport[T io.Closer](closer T, logger *log.Logger) {
	err := closer.Close()
	if err != nil {
		if logger == nil {
			logger = log.Default()
		}
		logger.Error("failed to close resource", "error", err)
	}
}
