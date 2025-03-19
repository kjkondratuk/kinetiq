package detection

type Watcher struct {
	EventsChan chan interface{}
	ErrorsChan chan error
}

// bridge converts a `chan T` into a `chan interface{}`
func bridge[T any](in chan T) chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for v := range in {
			out <- v
		}
	}()
	return out
}

func NewWatcher[T Detectable](events chan T, err chan error) Watcher {
	return Watcher{
		EventsChan: bridge(events),
		ErrorsChan: err,
	}
}
