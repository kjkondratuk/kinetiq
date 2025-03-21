package loader

import (
	"context"
	"fmt"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/kjkondratuk/kinetiq/plugin/functions"
	"github.com/tetratelabs/wazero"
	"log"
	"os"
	"sync"
)

type lazyReloader struct {
	Loader
	closeablePlugin
	mutex sync.Mutex
	path  string
}

func newReloader(path string) lazyReloader {
	return lazyReloader{path: path}
}

type closeablePlugin interface {
	Close(ctx context.Context) error
	v1.ModuleService
}

func initialize(ctx context.Context) *v1.ModuleServicePlugin {
	plugin, err := v1.NewModuleServicePlugin(ctx, v1.WazeroModuleConfig(
		wazero.NewModuleConfig().
			WithStartFunctions("_initialize", "_start"). // unclear why adding this made things work... It should be doing this anyway...
			WithStdout(os.Stdout).
			WithStderr(os.Stderr),
	))
	if err != nil {
		log.Fatal("Failed to setup plugin environment", err)
	}
	log.Printf("Plugin environment setup...\n")
	return plugin
}

func (r *lazyReloader) load(ctx context.Context) (closeablePlugin, error) {
	pl := initialize(ctx)

	// don't allow loading and retrieval at the same time
	r.mutex.Lock()
	l, err := pl.Load(ctx, r.path, functions.NewDefaultPluginFunctions())
	r.mutex.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	r.closeablePlugin = l
	return r.closeablePlugin, nil
}

func (r *lazyReloader) Get(ctx context.Context) (v1.ModuleService, error) {
	var plugin v1.ModuleService
	if r.closeablePlugin != nil {
		// don't allow loading and retrieval at the same time
		r.mutex.Lock()
		plugin = r.closeablePlugin
		r.mutex.Unlock()
	} else {
		var err error
		plugin, err = r.load(ctx)
		if err != nil {
			return nil, err
		}
	}

	return plugin, nil
}

func (r *lazyReloader) Reload(ctx context.Context) error {
	r.Resolve(ctx)

	// Close existing plugin if loaded, since we're reloading
	err := r.closeablePlugin.Close(ctx)
	if err != nil {
		return fmt.Errorf("failed to close plugin for reload: %w", err)
	}

	_, err = r.load(ctx)
	if err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}

	return nil
}

func (r *lazyReloader) Close(ctx context.Context) error {
	if r.closeablePlugin != nil {
		return r.closeablePlugin.Close(ctx)
	}
	return nil
}

type Loader interface {
	Resolve(ctx context.Context)
	Get(ctx context.Context) (v1.ModuleService, error)
	Reload(ctx context.Context) error
	Close(ctx context.Context) error
}
