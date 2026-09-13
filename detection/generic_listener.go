package detection

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"log"
	"sync"
	"time"
)

type listener[T Detectable] struct {
	watcher Watcher[T]
}

func NewListener[T Detectable](watcher Watcher[T]) Listener[T] {
	return &listener[T]{watcher: watcher}
}

// DefaultReloadDebounce is how long the watcher waits for filesystem activity to
// settle before signalling a reload. A single `go build` emits several
// Write/Create events; without this, each one would trigger a separate swap.
const DefaultReloadDebounce = 300 * time.Millisecond

// FilesystemNotificationReloadSignaller reports module changes by signalling
// reloadCh rather than reloading directly. The processor owns the actual swap so
// it can run between records, when no module is in use.
func FilesystemNotificationReloadSignaller(reloadCh chan<- struct{}, debounce time.Duration) Responder[fsnotify.Event] {
	var mu sync.Mutex
	var timer *time.Timer

	return func(notification *fsnotify.Event, err error) {
		if err != nil {
			log.Printf("Failed to handle file watcher changes: %s", err)
			return
		}
		if notification == nil {
			return
		}
		if !notification.Op.Has(fsnotify.Write) && !notification.Op.Has(fsnotify.Create) {
			return
		}

		log.Printf("Detected change in %s", notification.Name)

		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(debounce, func() {
			// Non-blocking: if a reload is already pending, this change is
			// covered by it.
			select {
			case reloadCh <- struct{}{}:
			default:
			}
		})
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
