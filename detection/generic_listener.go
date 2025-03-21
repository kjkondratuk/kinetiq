package detection

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"github.com/kjkondratuk/kinetiq/loader"
	"log"
)

type listener[T Detectable] struct {
	watcher Watcher[T]
}

func NewListener[T Detectable](watcher Watcher[T]) Listener[T] {
	return &listener[T]{watcher: watcher}
}

func FilesystemNotificationPluginReloadResponder(ctx context.Context, dl loader.Loader) Responder[fsnotify.Event] {
	return func(notification *fsnotify.Event, err error) {
		if err != nil {
			log.Printf("Failed to handle file watcher changes: %s", err)
			return
		}
		if notification.Op.Has(fsnotify.Write) || notification.Op.Has(fsnotify.Create) {
			log.Printf("Detected change in %s", notification.Name)
			err = dl.Reload(ctx)
			if err != nil {
				log.Printf("Failed to reload plugin: %s", err)
				return
			}
		}
	}
}

func (f *listener[T]) Listen(ctx context.Context, responder Responder[T]) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-f.watcher.EventsChan():
			if !ok {
				return
			}
			log.Printf("event: %+v", event)
			responder(&event, nil)
		case err, ok := <-f.watcher.ErrorsChan():
			if !ok {
				return
			}
			log.Printf("error in listener: err: %s", err)
			responder(nil, err)
		}
	}
}
