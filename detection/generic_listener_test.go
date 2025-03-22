package detection

import (
	"context"
	"errors"
	"github.com/fsnotify/fsnotify"
	"github.com/kjkondratuk/kinetiq/loader"
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

func TestFilesystemNotificationPluginReloadResponder(t *testing.T) {
	type constructArgs struct {
		ctx context.Context
		dl  func() *loader.MockLoader
	}
	type callArgs struct {
		event *fsnotify.Event
		err   error
	}
	tests := []struct {
		name          string
		constructArgs constructArgs
		callArgs      callArgs
		validate      func(t *testing.T, mockLoader *loader.MockLoader)
	}{
		{
			"should handle error from event source",
			constructArgs{
				ctx: t.Context(),
				dl: func() *loader.MockLoader {
					ldr := &loader.MockLoader{}
					return ldr
				},
			},
			callArgs{
				event: nil,
				err:   errors.New("something went wrong"),
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertNotCalled(t, "Get", mock.Anything)
			},
		}, {
			"should handle non-write non-create events",
			constructArgs{
				ctx: t.Context(),
				dl: func() *loader.MockLoader {
					ldr := &loader.MockLoader{}
					return ldr
				},
			},
			callArgs{
				event: &fsnotify.Event{
					Name: "some_file",
					Op:   fsnotify.Remove,
				},
				err: nil,
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertNotCalled(t, "Get", mock.Anything)
			},
		}, {
			"should handle errors reloading gracefully",
			constructArgs{
				ctx: t.Context(),
				dl: func() *loader.MockLoader {
					ldr := &loader.MockLoader{}
					ldr.On("Reload", mock.Anything).Return(errors.New("something bad happened"))
					return ldr
				},
			},
			callArgs{
				event: &fsnotify.Event{
					Name: "some_file",
					Op:   fsnotify.Create,
				},
				err: nil,
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertCalled(t, "Reload", mock.Anything)
			},
		}, {
			"should handle successful reloading for create events",
			constructArgs{
				ctx: t.Context(),
				dl: func() *loader.MockLoader {
					ldr := &loader.MockLoader{}
					ldr.On("Reload", mock.Anything).Return(nil)
					return ldr
				},
			},
			callArgs{
				event: &fsnotify.Event{
					Name: "some_file",
					Op:   fsnotify.Create,
				},
				err: nil,
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertCalled(t, "Reload", mock.Anything)
			},
		}, {
			"should handle successful reloading for write events",
			constructArgs{
				ctx: t.Context(),
				dl: func() *loader.MockLoader {
					ldr := &loader.MockLoader{}
					ldr.On("Reload", mock.Anything).Return(nil)
					return ldr
				},
			},
			callArgs{
				event: &fsnotify.Event{
					Name: "some_file",
					Op:   fsnotify.Write,
				},
				err: nil,
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertCalled(t, "Reload", mock.Anything)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ldr := tt.constructArgs.dl()
			responder := FilesystemNotificationPluginReloadResponder(tt.constructArgs.ctx, ldr)
			responder(tt.callArgs.event, tt.callArgs.err)
			tt.validate(t, ldr)
		})
	}
}
