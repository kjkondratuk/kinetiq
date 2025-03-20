package detection

import (
	"context"
	"github.com/fsnotify/fsnotify"
)

type Detectable interface {
	S3EventNotification | fsnotify.Event | MockEvent
}

// Mock implementation of Detectable for testing
type MockEvent struct {
	ID int
}

type Listener[T Detectable] interface {
	Listen(ctx context.Context, responder Responder[T])
}

type Responder[T Detectable] func(notification *T, err error)
