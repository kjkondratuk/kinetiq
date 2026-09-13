package loader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/stretchr/testify/require"
)

// TestGuestModuleSeesRealWallClock guards against wazero's default fake clock
// (fixed at 2022/01/01 00:00:00, advancing 1ms per reading -- see
// internal/platform/time.go and internal/sys/sys.go in wazero v1.9.0) leaking
// into guest modules. The example module logs via the standard `log` package
// in its Process method, which prefixes every line with a `YYYY/MM/DD
// HH:MM:SS` timestamp derived from the guest's view of wall time. Go's `log`
// package writes to os.Stderr by default, and loader.go wires the guest's
// WASI stderr to the host's os.Stderr via WithStderr(os.Stderr), so that's
// the stream we capture. `load` reads the os.Stderr package variable at call
// time, so we can replace it with a pipe, invoke the module, and inspect what
// the guest logged.
func TestGuestModuleSeesRealWallClock(t *testing.T) {
	if _, err := os.Stat(coexistModulePath); err != nil {
		t.Skip("run `make build-test-module` first; artifact is gitignored")
	}

	// Replace os.Stderr with a pipe so we can capture what the guest module
	// logs, and restore it unconditionally so a failure here can't corrupt
	// the test binary's stderr for other tests.
	r, w, err := os.Pipe()
	require.NoError(t, err, "creating pipe")
	origStderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
	}()

	// Read from the pipe in a goroutine so a full pipe buffer can't deadlock
	// the test while the module is writing to it.
	var buf bytes.Buffer
	var readErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, readErr = io.Copy(&buf, r)
	}()

	ctx := context.Background()
	l := &defaultPluginLoader{}
	var mu sync.Mutex

	plugin, err := l.load(ctx, &mu, coexistModulePath)
	require.NoError(t, err, "load must succeed")

	req := &v1.ProcessRequest{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}
	_, procErr := plugin.Process(ctx, req)

	// Close the plugin, then the pipe writer, before reading the captured
	// output -- otherwise the reader goroutine never sees EOF.
	closeErr := plugin.Close(ctx)
	writeCloseErr := w.Close()

	<-done

	require.NoError(t, procErr, "Process must succeed")
	require.NoError(t, closeErr, "plugin Close must succeed")
	require.NoError(t, writeCloseErr, "pipe writer Close must succeed")
	require.NoError(t, readErr, "reading captured stderr must succeed")

	output := buf.String()
	t.Logf("captured guest stderr:\n%s", output)

	expectedYear := fmt.Sprintf("%d/", time.Now().Year())
	require.Contains(t, output, expectedYear,
		"guest module sees wazero's fake clock; expected real wall time (current year %d) in log output, got: %s",
		time.Now().Year(), output)

	require.False(t, strings.Contains(output, "2022/01/01"),
		"guest module sees wazero's fake clock; expected real wall time, but got frozen 2022/01/01 date in log output: %s",
		output)
}
