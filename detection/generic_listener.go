package detection

import (
	"fmt"
	"log"
)

type listener[T Detectable] struct {
	watcher Watcher
	path    *string
}

func NewListener[T Detectable](watcher Watcher) Listener[T] {
	return &listener[T]{watcher: watcher}
}

func (f *listener[T]) Listen(responder Responder[T]) {
	for {
		select {
		case event, ok := <-f.watcher.EventsChan:
			log.Printf("event: %s", event)
			if !ok {
				return
			}
			log.Printf("code change event: %+v", event)
			ev, ok := event.(T)
			if !ok {
				responder(nil, fmt.Errorf("event is not of expected type: %T", ev))
				continue
			}
			responder(&ev, nil)
		case err, ok := <-f.watcher.ErrorsChan:
			if !ok {
				return
			}
			log.Printf("error watching file: err: %s", err)
			responder(nil, err)
		}
	}
}
