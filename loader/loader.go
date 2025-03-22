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

type defaultPluginLoader struct{}

type pluginLoader interface {
	load(ctx context.Context, mutex *sync.Mutex, path string) (closeablePlugin, error)
}

func (l *defaultPluginLoader) load(ctx context.Context, mutex *sync.Mutex, path string) (closeablePlugin, error) {
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
	Loader
	closeablePlugin
	pluginLoader
	mutex sync.Mutex
	path  string
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

func (r *lazyReloader) Resolve(ctx context.Context) {
	// empty because we don't need to resolve local files
	fmt.Println("Resolving local file")
}

func (r *lazyReloader) Get(ctx context.Context) (v1.ModuleService, error) {
	var plugin closeablePlugin
	if r.closeablePlugin != nil {
		// don't allow loading and retrieval at the same time
		r.mutex.Lock()
		plugin = r.closeablePlugin
		r.mutex.Unlock()
	} else {
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
	r.Resolve(ctx)

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
	Resolve(ctx context.Context)
	Get(ctx context.Context) (v1.ModuleService, error)
	Reload(ctx context.Context) error
	Close(ctx context.Context) error
}
