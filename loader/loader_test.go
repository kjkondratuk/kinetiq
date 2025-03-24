package loader

import (
	"context"
	"errors"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
)

func TestLazyReloader_Resolve(t *testing.T) {
	// Ensure Resolve doesn't panic and runs correctly
	reloader := lazyReloader{path: "dummy-path"}
	reloader.Resolve(context.Background())
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

func TestLazyReloader_Reload(t *testing.T) {
	t.Run("successful_reload", func(t *testing.T) {
		ctx := t.Context()
		mockCloseablePlugin := &MockcloseablePlugin{}
		mockLoader := &MockpluginLoader{}

		reloader := lazyReloader{
			path:            "dummy-path",
			closeablePlugin: mockCloseablePlugin,
			pluginLoader:    mockLoader,
			mutex:           sync.Mutex{},
		}

		mockCloseablePlugin.On("Close", ctx).Return(nil)
		newMockPlugin := &MockcloseablePlugin{}
		mockLoader.On("load", ctx, mock.Anything, "dummy-path").Return(newMockPlugin, nil)

		err := reloader.Reload(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if reloader.closeablePlugin != newMockPlugin {
			t.Fatalf("expected closeablePlugin to be %v, got %v", newMockPlugin, reloader.closeablePlugin)
		}

		mockCloseablePlugin.AssertExpectations(t)
		mockLoader.AssertExpectations(t)
	})

	t.Run("failure_on_close", func(t *testing.T) {
		ctx := t.Context()
		mockCloseablePlugin := &MockcloseablePlugin{}
		reloader := lazyReloader{
			path:            "dummy-path",
			closeablePlugin: mockCloseablePlugin,
			mutex:           sync.Mutex{},
		}

		mockCloseablePlugin.On("Close", ctx).Return(errors.New("close error"))

		err := reloader.Reload(ctx)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("failure_on_load", func(t *testing.T) {
		ctx := t.Context()
		mockCloseablePlugin := &MockcloseablePlugin{}
		mockLoader := &MockpluginLoader{}

		reloader := lazyReloader{
			path:            "dummy-path",
			closeablePlugin: mockCloseablePlugin,
			pluginLoader:    mockLoader,
			mutex:           sync.Mutex{},
		}

		mockCloseablePlugin.On("Close", ctx).Return(nil)
		mockLoader.On("load", ctx, mock.Anything, "dummy-path").Return(nil, errors.New("load error"))

		err := reloader.Reload(ctx)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
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
