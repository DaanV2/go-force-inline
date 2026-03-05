package component_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/daanv2/go-force-inline/pkg/profgen"
	"github.com/daanv2/go-force-inline/pkg/resolver"
	"github.com/daanv2/go-force-inline/pkg/verifier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndVerify(t *testing.T) {
	outDir := t.TempDir()
	outputPath := filepath.Join(outDir, "default.pgo")

	// Step 1: Resolve directives from testdata
	edges, err := resolver.Resolve([]string{"../testdata/"})
	require.NoError(t, err)
	require.Len(t, edges, 2)

	// Verify edge contents
	for _, edge := range edges {
		assert.Contains(t, edge.CallerName, ".handler")
		assert.Positive(t, edge.CallSiteOffset)
	}

	// Check first edge is processRequest with weight=10000
	assert.Contains(t, edges[0].CalleeName, ".processRequest")
	assert.Equal(t, int64(10000), edges[0].Weight)

	// Check second edge is validateInput with default weight
	assert.Contains(t, edges[1].CalleeName, ".validateInput")
	assert.Equal(t, int64(10000), edges[1].Weight)

	// Step 2: Generate profile
	err = profgen.Generate(edges, outputPath)
	require.NoError(t, err)

	// Verify the file was created
	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Positive(t, info.Size())

	// Step 3: Verify the profile
	var buf bytes.Buffer
	err = verifier.Verify(outputPath, 99.0, false, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "yes")
	assert.Contains(t, output, "processRequest")
	assert.Contains(t, output, "validateInput")
	assert.Contains(t, output, "Total weight: 20000")
}
