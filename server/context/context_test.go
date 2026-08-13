package context

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/root-gg/plik/server/common"
	"github.com/stretchr/testify/require"
)

func newTestContext() *Context {
	return &Context{config: common.NewConfiguration()}
}

// TestGenContextUpToDate ensures that gen.go output matches context.go.
// If this test fails, run: go run gen.go > context.go
func TestGenContextUpToDate(t *testing.T) {
	// Run gen.go and capture output
	cmd := exec.Command("go", "run", "gen.go")
	out, err := cmd.Output()
	require.NoError(t, err, "gen.go failed to run: %s", err)

	// Read current context.go
	existing, err := os.ReadFile("context.go")
	require.NoError(t, err, "unable to read context.go")

	// Compare (trim trailing whitespace for robustness)
	require.Equal(t,
		strings.TrimRight(string(existing), "\n"),
		strings.TrimRight(string(out), "\n"),
		"context.go is out of date — run: go run gen.go > context.go",
	)
}
