package loader

import (
	"context"
	"fmt"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/kjkondratuk/kinetiq/plugin/functions"
	"github.com/tetratelabs/wazero"
	"log"
	"log/slog"
	"os"
	"sync"
)

type defaultPluginLoader struct{}

type pluginLoader interface {
	load(ctx context.Context, mutex *sync.Mutex, path string) (closeablePlugin, error)
}

func (l *defaultPluginLoader) load(ctx context.Context, mutex *sync.Mutex, path string) (closeablePlugin, error) {
	// TODO : introduce a filesystem abstraction so we can more easily unit test this perhaps
	plugin, err := v1.NewModuleServicePlugin(ctx, v1.WazeroModuleConfig(
		wazero.NewModuleConfig().
			WithStartFunctions("_initialize", "_start"). // unclear why adding this made things work... It should be doing this anyway...
			WithStdout(os.Stdout).
			WithStderr(os.Stderr),
	))
	if err != nil {
		slog.Error("Failed to setup plugin environment", slog.String("err", err.Error()))
	}
	log.Printf("Plugin environment setup...\n")

	// don't allow loading and retrieval at the same time
	mutex.Lock()
	ld, err := plugin.Load(ctx, path, functions.NewDefaultPluginFunctions())
	mutex.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	return ld, nil
}

type lazyReloader struct {
	closeablePlugin
	pluginLoader
	mutex sync.Mutex
	path  string

	// resolve fetches the module artifact to path before it is loaded. It is nil
	// for loaders whose artifact is already present on local disk.
	//
	// This is a function field rather than a method on an embedded type on
	// purpose. Go selects methods at compile time by embedding depth, so a
	// Resolve method declared on lazyReloader always shadows one declared on a
	// type that embeds lazyReloader. Dispatching through an embedded interface
	// does not change that, which previously left the S3 implementation
	// unreachable.
	resolve func(ctx context.Context) error
}

func newReloader(path string) lazyReloader {
	return lazyReloader{path: path, pluginLoader: &defaultPluginLoader{}}
}

func NewBasicReloader(path string) Loader {
	r := newReloader(path)
	return &r
}

type closeablePlugin interface {
	Close(ctx context.Context) error
	v1.ModuleService
}

// resolveArtifact runs the configured resolver, if one is set.
func (r *lazyReloader) resolveArtifact(ctx context.Context) error {
	if r.resolve == nil {
		slog.Info("Using local artifact", slog.String("path", r.path))
		return nil
	}
	return r.resolve(ctx)
}

func (r *lazyReloader) Get(ctx context.Context) (v1.ModuleService, error) {
	var plugin closeablePlugin
	if r.closeablePlugin != nil {
		// don't allow loading and retrieval at the same time
		r.mutex.Lock()
		plugin = r.closeablePlugin
		r.mutex.Unlock()
	} else {
		if err := r.resolveArtifact(ctx); err != nil {
			return nil, fmt.Errorf("failed to resolve plugin artifact: %w", err)
		}

		var err error
		plugin, err = r.load(ctx, &r.mutex, r.path)
		if err != nil {
			return nil, err
		}
		r.closeablePlugin = plugin
	}

	return plugin, nil
}

func (r *lazyReloader) Reload(ctx context.Context) error {
	// Resolve before tearing anything down: if the new artifact cannot be
	// fetched, keep serving the module that is already loaded.
	if err := r.resolveArtifact(ctx); err != nil {
		return fmt.Errorf("failed to resolve plugin artifact for reload: %w", err)
	}

	// Close existing plugin if loaded, since we're reloading
	err := r.Close(ctx)
	if err != nil {
		return fmt.Errorf("failed to close plugin for reload: %w", err)
	}

	ld, err := r.load(ctx, &r.mutex, r.path)
	if err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}

	r.closeablePlugin = ld

	return nil
}

func (r *lazyReloader) Close(ctx context.Context) error {
	if r.closeablePlugin != nil {
		return r.closeablePlugin.Close(ctx)
	}
	return nil
}

type Loader interface {
	Get(ctx context.Context) (v1.ModuleService, error)
	Reload(ctx context.Context) error
	Close(ctx context.Context) error
}
