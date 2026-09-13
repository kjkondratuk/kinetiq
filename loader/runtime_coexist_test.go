package loader

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const coexistModulePath = "../examples/test_module/test_module.wasm"

// The atomic-swap design loads a replacement module while the previous one is
// still open. This test exists to prove wazero tolerates that.
func TestTwoRuntimesCoexist(t *testing.T) {
	if _, err := os.Stat(coexistModulePath); err != nil {
		t.Skip("run `make build-test-module` first; artifact is gitignored")
	}

	ctx := context.Background()
	l := &defaultPluginLoader{}
	var mu sync.Mutex

	first, err := l.load(ctx, &mu, coexistModulePath)
	require.NoError(t, err, "first load must succeed")
	defer first.Close(ctx)

	second, err := l.load(ctx, &mu, coexistModulePath)
	require.NoError(t, err, "second load must succeed while the first is still open")
	defer second.Close(ctx)

	assert.NotSame(t, first, second, "loads must produce distinct plugin instances")
}
