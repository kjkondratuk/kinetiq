package loader

import (
	"context"
	"fmt"
)

type filesystemLoader struct {
	lazyReloader
}

func NewFilesystemLoader(pluginRef string) Loader {
	l := &filesystemLoader{lazyReloader: newReloader(pluginRef)}
	l.Loader = l
	return l
}

func (l *filesystemLoader) Reload(ctx context.Context) error {
	return l.lazyReloader.Reload(ctx)
}

func (l *filesystemLoader) Resolve(ctx context.Context) {
	// empty because we don't need to resolve local files
	fmt.Println("Resolving local file")
}
