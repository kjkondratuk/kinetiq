package detection

type watcher[T Detectable] struct {
	eventsChan chan T
	errorsChan chan error
}

type Watcher[T Detectable] interface {
	EventsChan() chan T
	ErrorsChan() chan error
}

func NewWatcher[T Detectable](events chan T, err chan error) Watcher[T] {
	return &watcher[T]{
		eventsChan: events,
		errorsChan: err,
	}
}

func (w watcher[T]) EventsChan() chan T {
	return w.eventsChan
}

func (w watcher[T]) ErrorsChan() chan error {
	return w.errorsChan
}
