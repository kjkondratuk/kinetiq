package loader

import (
	"context"
	"errors"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLazyReloader_resolveArtifact(t *testing.T) {
	t.Run("no_resolver_is_a_noop", func(t *testing.T) {
		// A local-file loader has nothing to fetch and must not error.
		reloader := lazyReloader{path: "dummy-path"}
		assert.NoError(t, reloader.resolveArtifact(context.Background()))
	})

	t.Run("configured_resolver_runs", func(t *testing.T) {
		called := false
		reloader := lazyReloader{path: "dummy-path", resolve: func(ctx context.Context) error {
			called = true
			return nil
		}}

		assert.NoError(t, reloader.resolveArtifact(context.Background()))
		assert.True(t, called, "Configured resolver must be invoked")
	})

	t.Run("resolver_error_propagates", func(t *testing.T) {
		reloader := lazyReloader{path: "dummy-path", resolve: func(ctx context.Context) error {
			return errors.New("boom")
		}}

		assert.ErrorContains(t, reloader.resolveArtifact(context.Background()), "boom")
	})
}

func TestLazyReloader_Get(t *testing.T) {
	t.Run("with_existing_plugin", func(t *testing.T) {
		ctx := t.Context()

		mockCloseablePlugin := &MockcloseablePlugin{}
		mockLoader := &MockpluginLoader{}
		reloader := lazyReloader{
			path:            "dummy-path",
			pluginLoader:    mockLoader,
			closeablePlugin: nil,
			mutex:           sync.Mutex{},
		}
		reloader.closeablePlugin = mockCloseablePlugin

		//mockCloseablePlugin.On("Process", ctx, mock.Anything).Return(nil, nil)

		plugin, err := reloader.Get(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if plugin != mockCloseablePlugin {
			t.Fatalf("expected plugin to be %v, got %v", mockCloseablePlugin, plugin)
		}

		mockLoader.AssertNotCalled(t, "load", ctx, mock.Anything, "dummy-path")
		mockCloseablePlugin.AssertExpectations(t)
	})

	t.Run("without_existing_plugin", func(t *testing.T) {
		ctx := t.Context()

		mockLoader := &MockpluginLoader{}
		reloader := lazyReloader{
			path:            "dummy-path",
			pluginLoader:    mockLoader,
			closeablePlugin: nil,
			mutex:           sync.Mutex{},
		}
		mockPlugin := &MockcloseablePlugin{}

		mockLoader.On("load", ctx, mock.Anything, "dummy-path").Return(mockPlugin, nil)

		reloader.closeablePlugin = nil

		plugin, err := reloader.Get(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if plugin != mockPlugin {
			t.Fatalf("expected plugin to be %v, got %v", mockPlugin, plugin)
		}

		mockLoader.AssertExpectations(t)
	})
}

func TestLazyReloader_Close(t *testing.T) {
	ctx := context.Background()

	t.Run("with_plugin", func(t *testing.T) {
		mockCloseablePlugin := &MockcloseablePlugin{}
		reloader := lazyReloader{closeablePlugin: mockCloseablePlugin}

		mockCloseablePlugin.On("Close", ctx).Return(nil)

		err := reloader.Close(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		mockCloseablePlugin.AssertExpectations(t)
	})

	t.Run("without_plugin", func(t *testing.T) {
		reloader := lazyReloader{closeablePlugin: nil}

		err := reloader.Close(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestLazyReloader_Close_ClearsPlugin(t *testing.T) {
	t.Run("close_nils_field_so_get_reloads", func(t *testing.T) {
		ctx := t.Context()
		closed := &MockcloseablePlugin{}
		closed.On("Close", ctx).Return(nil)

		reloader := lazyReloader{
			path:            "dummy-path",
			closeablePlugin: closed,
			pluginLoader:    &MockpluginLoader{},
			mutex:           sync.Mutex{},
		}

		assert.NoError(t, reloader.Close(ctx))
		assert.Nil(t, reloader.closeablePlugin,
			"Close must clear the reference or Get will serve a closed plugin")
		closed.AssertExpectations(t)
	})
}

func TestLazyReloader_Reload_Atomic(t *testing.T) {
	t.Run("failed_load_keeps_previous_module_serving", func(t *testing.T) {
		ctx := t.Context()
		original := &MockcloseablePlugin{}
		mockLoader := &MockpluginLoader{}
		mockLoader.On("load", ctx, mock.Anything, "dummy-path").
			Return(nil, errors.New("load error"))

		reloader := lazyReloader{
			path:            "dummy-path",
			closeablePlugin: original,
			pluginLoader:    mockLoader,
			mutex:           sync.Mutex{},
		}

		err := reloader.Reload(ctx)

		assert.ErrorContains(t, err, "load error")
		assert.Same(t, original, reloader.closeablePlugin,
			"a failed load must leave the previous module installed")
		original.AssertNotCalled(t, "Close", ctx)
		mockLoader.AssertExpectations(t)
	})

	t.Run("successful_reload_closes_only_the_old_plugin", func(t *testing.T) {
		ctx := t.Context()
		original := &MockcloseablePlugin{}
		replacement := &MockcloseablePlugin{}
		mockLoader := &MockpluginLoader{}

		original.On("Close", ctx).Return(nil)
		mockLoader.On("load", ctx, mock.Anything, "dummy-path").Return(replacement, nil)

		reloader := lazyReloader{
			path:            "dummy-path",
			closeablePlugin: original,
			pluginLoader:    mockLoader,
			mutex:           sync.Mutex{},
		}

		assert.NoError(t, reloader.Reload(ctx))
		assert.Same(t, replacement, reloader.closeablePlugin)
		replacement.AssertNotCalled(t, "Close", ctx)
		original.AssertExpectations(t)
	})

	t.Run("close_error_on_old_plugin_does_not_fail_reload", func(t *testing.T) {
		ctx := t.Context()
		original := &MockcloseablePlugin{}
		replacement := &MockcloseablePlugin{}
		mockLoader := &MockpluginLoader{}

		original.On("Close", ctx).Return(errors.New("close error"))
		mockLoader.On("load", ctx, mock.Anything, "dummy-path").Return(replacement, nil)

		reloader := lazyReloader{
			path:            "dummy-path",
			closeablePlugin: original,
			pluginLoader:    mockLoader,
			mutex:           sync.Mutex{},
		}

		assert.NoError(t, reloader.Reload(ctx),
			"swap already succeeded; a close failure is logged, not returned")
		assert.Same(t, replacement, reloader.closeablePlugin)
	})

	t.Run("after_close_does_not_resurrect_a_plugin", func(t *testing.T) {
		ctx := t.Context()
		mockLoader := &MockpluginLoader{}
		loaded := &MockcloseablePlugin{}
		loaded.On("Close", ctx).Return(nil)
		mockLoader.On("load", ctx, mock.Anything, "dummy-path").Return(loaded, nil)

		reloader := lazyReloader{
			path:         "dummy-path",
			pluginLoader: mockLoader,
			mutex:        sync.Mutex{},
		}

		require.NoError(t, reloader.Close(ctx))

		err := reloader.Reload(ctx)

		assert.ErrorContains(t, err, "loader is closed")
		assert.Nil(t, reloader.closeablePlugin,
			"a reload racing a close must not install a plugin nothing will ever close")
		loaded.AssertExpectations(t)
	})
}

func TestLazyReloader_Get_ClosedGuardAndDoubleCheck(t *testing.T) {
	t.Run("after_close_returns_error_and_closes_the_loaded_plugin", func(t *testing.T) {
		ctx := t.Context()
		mockLoader := &MockpluginLoader{}
		loaded := &MockcloseablePlugin{}
		loaded.On("Close", ctx).Return(nil)
		mockLoader.On("load", ctx, mock.Anything, "dummy-path").Return(loaded, nil)

		reloader := lazyReloader{
			path:         "dummy-path",
			pluginLoader: mockLoader,
			mutex:        sync.Mutex{},
		}

		require.NoError(t, reloader.Close(ctx))

		plugin, err := reloader.Get(ctx)

		assert.Nil(t, plugin)
		assert.ErrorContains(t, err, "loader is closed")
		assert.Nil(t, reloader.closeablePlugin)
		loaded.AssertExpectations(t)
	})

	t.Run("plugin_installed_while_loading_is_preferred_and_ours_is_closed", func(t *testing.T) {
		ctx := t.Context()
		installed := &MockcloseablePlugin{}
		loadedButRedundant := &MockcloseablePlugin{}
		loadedButRedundant.On("Close", ctx).Return(nil)

		reloader := &lazyReloader{path: "dummy-path", mutex: sync.Mutex{}}

		mockLoader := &MockpluginLoader{}
		mockLoader.On("load", ctx, mock.Anything, "dummy-path").
			Run(func(args mock.Arguments) {
				// Simulate another goroutine installing a plugin while this
				// call was loading its own -- the double-check in Get must
				// prefer the already-installed plugin and close ours.
				reloader.mutex.Lock()
				reloader.closeablePlugin = installed
				reloader.mutex.Unlock()
			}).
			Return(loadedButRedundant, nil)
		reloader.pluginLoader = mockLoader

		plugin, err := reloader.Get(ctx)

		assert.NoError(t, err)
		assert.Same(t, installed, plugin)
		assert.Same(t, installed, reloader.closeablePlugin)
		loadedButRedundant.AssertExpectations(t)
	})
}

// raceTestPlugin and raceTestLoader are minimal, allocation-cheap fakes (not
// mockery mocks) used purely to hammer lazyReloader with real concurrent
// goroutines under `go test -race`. They carry no expectations of their own;
// their only job is to exist as distinct instances so the race detector can
// see genuine unsynchronized access to lazyReloader's fields.
type raceTestPlugin struct {
	closed int32
}

func (p *raceTestPlugin) Close(_ context.Context) error {
	atomic.AddInt32(&p.closed, 1)
	return nil
}

func (p *raceTestPlugin) Process(_ context.Context, _ *v1.ProcessRequest) (*v1.ProcessResponse, error) {
	return &v1.ProcessResponse{}, nil
}

type raceTestLoader struct{}

func (l *raceTestLoader) load(_ context.Context, _ *sync.Mutex, _ string) (closeablePlugin, error) {
	// A tiny sleep widens the window in which Get/Reload/Close can interleave,
	// making a genuine race far more likely to be observed within the test's
	// short run time.
	time.Sleep(time.Millisecond)
	return &raceTestPlugin{}, nil
}

// TestLazyReloader_ConcurrentGetReloadClose_NoRace drives Get, Reload, and
// Close from separate goroutines simultaneously. It exists to give `go test
// -race` real concurrent access to lazyReloader's fields: Get's unsynchronized
// read/write of closeablePlugin was a genuine data race with the locked
// access in Reload/Close, and this test is what catches it.
func TestLazyReloader_ConcurrentGetReloadClose_NoRace(t *testing.T) {
	ctx := context.Background()
	reloader := &lazyReloader{
		path:         "dummy-path",
		pluginLoader: &raceTestLoader{},
		mutex:        sync.Mutex{},
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = reloader.Get(ctx)
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 30; i++ {
			_ = reloader.Reload(ctx)
		}
	}()

	time.Sleep(20 * time.Millisecond)
	_ = reloader.Close(ctx)

	close(stop)
	wg.Wait()
}
