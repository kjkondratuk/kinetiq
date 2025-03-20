package detection

import (
	"errors"
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
