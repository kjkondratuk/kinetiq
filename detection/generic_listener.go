package detection

import (
	"context"
	"log"
)

type listener[T Detectable] struct {
	watcher Watcher[T]
}

func NewListener[T Detectable](watcher Watcher[T]) Listener[T] {
	return &listener[T]{watcher: watcher}
}

func (f *listener[T]) Listen(ctx context.Context, responder Responder[T]) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-f.watcher.EventsChan():
			log.Printf("event: %+v", event)
			if !ok {
				return
			}
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
