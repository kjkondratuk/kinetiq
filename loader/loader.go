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
			WithStderr(os.Stderr).
			// wazero defaults to a fake clock fixed at 2022-01-01 that advances
			// 1ms per reading. Guest modules legitimately need real time for
			// windowing, TTLs and record timestamps, so supply the host's.
			WithSysWalltime().
			WithSysNanotime().
			WithSysNanosleep(),
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
	mutex  sync.Mutex
	path   string
	closed bool

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
	r.mutex.Lock()
	plugin := r.closeablePlugin
	r.mutex.Unlock()

	if plugin != nil {
		return plugin, nil
	}

	if err := r.resolveArtifact(ctx); err != nil {
		return nil, fmt.Errorf("failed to resolve plugin artifact: %w", err)
	}

	// Load without holding the mutex: load() acquires it internally, and
	// holding it here would deadlock.
	loaded, err := r.load(ctx, &r.mutex, r.path)
	if err != nil {
		return nil, err
	}

	r.mutex.Lock()
	switch {
	case r.closed:
		// A Close() landed while we were loading. Nothing will ever close
		// this plugin if we install it, so close it ourselves and report
		// that the loader is no longer usable.
		r.mutex.Unlock()
		if cerr := loaded.Close(ctx); cerr != nil {
			slog.Error("failed to close plugin loaded after Close", slog.String("err", cerr.Error()))
		}
		return nil, fmt.Errorf("loader is closed")
	case r.closeablePlugin != nil:
		// Another goroutine's Get (or a Reload) installed a plugin while we
		// were loading ours. Prefer the already-installed one and close the
		// redundant one we just loaded.
		existing := r.closeablePlugin
		r.mutex.Unlock()
		if cerr := loaded.Close(ctx); cerr != nil {
			slog.Error("failed to close redundant plugin", slog.String("err", cerr.Error()))
		}
		return existing, nil
	default:
		r.closeablePlugin = loaded
		r.mutex.Unlock()
		return loaded, nil
	}
}

func (r *lazyReloader) Reload(ctx context.Context) error {
	// Resolve before tearing anything down: if the new artifact cannot be
	// fetched, keep serving the module that is already loaded.
	if err := r.resolveArtifact(ctx); err != nil {
		return fmt.Errorf("failed to resolve plugin artifact for reload: %w", err)
	}

	// Load the replacement while the current module keeps serving. A failure
	// here must leave the running module untouched.
	ld, err := r.load(ctx, &r.mutex, r.path)
	if err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}

	r.mutex.Lock()
	if r.closed {
		// A Close() raced us and won: the loader is shutting down, so don't
		// resurrect it by installing the plugin we just loaded. Close it
		// instead -- otherwise nothing ever would.
		r.mutex.Unlock()
		if cerr := ld.Close(ctx); cerr != nil {
			slog.Error("failed to close plugin loaded after Close", slog.String("err", cerr.Error()))
		}
		return fmt.Errorf("loader is closed")
	}
	old := r.closeablePlugin
	r.closeablePlugin = ld
	r.mutex.Unlock()

	// The swap has already succeeded, so a close failure is logged rather than
	// returned -- the caller has a working module either way.
	if old != nil {
		if cerr := old.Close(ctx); cerr != nil {
			slog.Error("failed to close previous plugin after swap", slog.String("err", cerr.Error()))
		}
	}

	return nil
}

func (r *lazyReloader) Close(ctx context.Context) error {
	r.mutex.Lock()
	p := r.closeablePlugin
	r.closeablePlugin = nil
	r.closed = true
	r.mutex.Unlock()

	if p != nil {
		return p.Close(ctx)
	}
	return nil
}

type Loader interface {
	Get(ctx context.Context) (v1.ModuleService, error)
	Reload(ctx context.Context) error
	Close(ctx context.Context) error
}
