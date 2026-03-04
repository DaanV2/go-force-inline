package xio_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/DaanV2/go-chess/pkg/extensions/xio"
	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/require"
)

type mockCloser struct {
	closed     bool
	closeErr   error
	closeCalls int
}

func (m *mockCloser) Close() error {
	m.closeCalls++
	m.closed = true

	return m.closeErr
}

func Test_CloseReport(t *testing.T) {
	t.Run("close without error", func(t *testing.T) {
		// Arrange
		closer := &mockCloser{}
		var buf bytes.Buffer
		logger := log.NewWithOptions(&buf, log.Options{Level: log.ErrorLevel})

		// Act
		xio.CloseReport(closer, logger)

		// Assert
		require.True(t, closer.closed)
		require.Equal(t, 1, closer.closeCalls)
		require.Empty(t, buf.String())
	})

	t.Run("close with error logs error", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("close error")
		closer := &mockCloser{closeErr: expectedErr}
		var buf bytes.Buffer
		logger := log.NewWithOptions(&buf, log.Options{Level: log.ErrorLevel})

		// Act
		xio.CloseReport(closer, logger)

		// Assert
		require.True(t, closer.closed)
		require.Equal(t, 1, closer.closeCalls)
		require.NotEmpty(t, buf.String())
		require.Contains(t, buf.String(), "failed to close resource")
	})

	t.Run("close with nil logger uses default", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("close error")
		closer := &mockCloser{closeErr: expectedErr}

		// Act & Assert (should not panic)
		require.NotPanics(t, func() {
			xio.CloseReport(closer, nil)
		})
		require.True(t, closer.closed)
	})

	t.Run("close io.ReadCloser", func(t *testing.T) {
		// Arrange
		reader := io.NopCloser(bytes.NewReader([]byte("test")))
		var buf bytes.Buffer
		logger := log.NewWithOptions(&buf, log.Options{Level: log.ErrorLevel})

		// Act
		xio.CloseReport(reader, logger)

		// Assert - should close without error
		require.Empty(t, buf.String())
	})

	t.Run("close io.WriteCloser", func(t *testing.T) {
		// Arrange
		var buf bytes.Buffer
		writer := &testWriteCloser{Writer: &buf}
		logger := log.NewWithOptions(&buf, log.Options{Level: log.ErrorLevel})

		// Act
		xio.CloseReport(writer, logger)

		// Assert
		require.True(t, writer.closed)
	})
}

type testWriteCloser struct {
	io.Writer
	closed bool
}

func (w *testWriteCloser) Close() error {
	w.closed = true

	return nil
}
