package loader

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
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
}
