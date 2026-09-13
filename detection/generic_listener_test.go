package detection

import (
	"errors"
	"github.com/fsnotify/fsnotify"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockResponder is a responder implementation for table-driven tests
type MockResponder[T Detectable] struct {
	mock.Mock
}

func (m *MockResponder[T]) Call(notification *T, err error) {
	m.Called(notification, err)
}

func Test_listener_Listen(t *testing.T) {
	tests := []struct {
		name                string
		setupWatcher        func(events chan MockEvent, errs chan error)
		expectedResponder   func(mockResponder *MockResponder[MockEvent])
		expectedEventLogged string
	}{
		{
			name: "valid event",
			setupWatcher: func(events chan MockEvent, errs chan error) {
				go func() {
					events <- MockEvent{ID: 1}
					close(events)
				}()
			},
			expectedResponder: func(mockResponder *MockResponder[MockEvent]) {
				mockResponder.On("Call", &MockEvent{ID: 1}, nil).Once()
			},
		},
		{
			name: "error event",
			setupWatcher: func(events chan MockEvent, errs chan error) {
				go func() {
					errs <- errors.New("watcher error")
					close(errs)
				}()
			},
			expectedResponder: func(mockResponder *MockResponder[MockEvent]) {
				mockResponder.On("Call", (*MockEvent)(nil), mock.MatchedBy(func(err error) bool {
					return err != nil && err.Error() == "watcher error"
				})).Once()
			},
		},
		{
			name: "no event and no error",
			setupWatcher: func(events chan MockEvent, errs chan error) {
				close(events)
				close(errs)
			},
			expectedResponder: func(_ *MockResponder[MockEvent]) {
				// No calls expected
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			eventsChan := make(chan MockEvent)
			errorsChan := make(chan error)
			test.setupWatcher(eventsChan, errorsChan)

			watcher := NewWatcher[MockEvent](eventsChan, errorsChan)
			listener := NewListener[MockEvent](watcher).(*listener[MockEvent])

			mockResponder := &MockResponder[MockEvent]{}
			test.expectedResponder(mockResponder)

			go listener.Listen(t.Context(), mockResponder.Call)

			time.Sleep(100 * time.Millisecond) // Ensure all goroutines execute

			mockResponder.AssertExpectations(t)
		})
	}
}

func TestFilesystemNotificationReloadSignaller(t *testing.T) {
	t.Run("burst_of_events_yields_one_signal", func(t *testing.T) {
		reloadCh := make(chan struct{}, 1)
		responder := FilesystemNotificationReloadSignaller(reloadCh, 50*time.Millisecond)

		// One `go build` emits several Write/Create events in quick succession.
		for i := 0; i < 5; i++ {
			responder(&fsnotify.Event{Name: "mod.wasm", Op: fsnotify.Write}, nil)
		}

		select {
		case <-reloadCh:
		case <-time.After(time.Second):
			t.Fatal("expected exactly one reload signal, got none")
		}

		select {
		case <-reloadCh:
			t.Fatal("burst must collapse to a single signal")
		case <-time.After(200 * time.Millisecond):
		}
	})

	t.Run("ignores_non_write_ops", func(t *testing.T) {
		reloadCh := make(chan struct{}, 1)
		responder := FilesystemNotificationReloadSignaller(reloadCh, 50*time.Millisecond)

		responder(&fsnotify.Event{Name: "mod.wasm", Op: fsnotify.Chmod}, nil)

		select {
		case <-reloadCh:
			t.Fatal("Chmod must not trigger a reload")
		case <-time.After(200 * time.Millisecond):
		}
	})

	t.Run("watcher_error_does_not_signal", func(t *testing.T) {
		reloadCh := make(chan struct{}, 1)
		responder := FilesystemNotificationReloadSignaller(reloadCh, 50*time.Millisecond)

		responder(nil, errors.New("watch failure"))

		select {
		case <-reloadCh:
			t.Fatal("an error must not trigger a reload")
		case <-time.After(200 * time.Millisecond):
		}
	})
}
