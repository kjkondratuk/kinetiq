package processor

import (
	"context"
	"errors"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/kjkondratuk/kinetiq/loader"
	"github.com/kjkondratuk/kinetiq/source"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWasmProcessor_Output(t *testing.T) {
	outputChannel := make(chan Result)
	p := &wasmProcessor{
		output: outputChannel,
	}

	assert.EqualValues(t, outputChannel, p.Output())
}

func TestWasmProcessor_Start(t *testing.T) {
	t.Run("process_success", func(t *testing.T) {
		mockModuleService := &v1.MockmoduleService{}
		mockLoader := &loader.MockLoader{}
		mockInputChannel := make(chan source.Record, 1)
		//outputChannel := make(chan Result, 1)

		ctx := context.Background()
		inputRecord := source.Record{
			Key:   []byte("input-key"),
			Value: []byte("input-value"),
			Headers: []source.RecordHeader{
				{Key: "header-key", Value: []byte("header-value")},
			},
			Ctx: ctx,
		}
		mockInputChannel <- inputRecord

		expectedResponse := &v1.ProcessResponse{
			Key:   []byte("processed-key"),
			Value: []byte("processed-value"),
			Headers: []*v1.Headers{
				{Key: "header-key", Value: []byte("header-value")},
			},
		}
		mockModuleService.On("Process", mock.Anything, mock.Anything).Return(expectedResponse, nil)
		mockLoader.On("Get", mock.Anything).Return(mockModuleService, nil)

		p, err := NewWasmProcessor(mockLoader, mockInputChannel, nil)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		go p.Start(ctx)

		result := <-p.Output()
		assert.Equal(t, expectedResponse.Key, result.Key)
		assert.Equal(t, expectedResponse.Value, result.Value)
		assert.Len(t, result.Headers, 1)
		assert.Equal(t, "header-key", result.Headers[0].Key)
		assert.Equal(t, []byte("header-value"), result.Headers[0].Value)
	})

	t.Run("process_error", func(t *testing.T) {
		mockModuleService := &v1.MockmoduleService{}
		mockLoader := &loader.MockLoader{}
		mockInputChannel := make(chan source.Record, 1)
		outputChannel := make(chan Result, 1)

		ctx := context.Background()
		inputRecord := source.Record{
			Key:   []byte("input-key"),
			Value: []byte("input-value"),
			Headers: []source.RecordHeader{
				{Key: "header-key", Value: []byte("header-value")},
			},
			Ctx: ctx,
		}
		mockInputChannel <- inputRecord

		mockModuleService.On("Process", mock.Anything, mock.Anything).Return(nil, errors.New("processing error"))
		mockLoader.On("Get", mock.Anything).Return(mockModuleService, nil)

		p, err := NewWasmProcessor(mockLoader, mockInputChannel, nil)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		go p.Start(ctx)

		select {
		case <-outputChannel:
			t.Fail() // Should not emit a result
		case <-time.After(200 * time.Millisecond):
			// No result emitted, expected behavior
		}
	})
}

func TestWasmProcessor_process(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockModuleService := &v1.MockmoduleService{}
		mockLoader := &loader.MockLoader{}

		ctx := context.Background()
		inputRecord := source.Record{
			Key:   []byte("input-key"),
			Value: []byte("input-value"),
			Headers: []source.RecordHeader{
				{Key: "header-key", Value: []byte("header-value")},
			},
			Ctx: ctx,
		}

		expectedResponse := &v1.ProcessResponse{
			Key:   []byte("processed-key"),
			Value: []byte("processed-value"),
			Headers: []*v1.Headers{
				{Key: "response-header-key", Value: []byte("response-header-value")},
			},
		}

		mockModuleService.On("Process", mock.Anything, mock.Anything).Return(expectedResponse, nil)
		mockLoader.On("Get", mock.Anything).Return(mockModuleService, nil)

		p := &wasmProcessor{
			ldr: mockLoader,
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		result, err := p.process(ctx, inputRecord)
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse.Key, result.Key)
		assert.Equal(t, expectedResponse.Value, result.Value)
		assert.Len(t, result.Headers, 1)
		assert.Equal(t, "response-header-key", result.Headers[0].Key)
		assert.Equal(t, []byte("response-header-value"), result.Headers[0].Value)
	})

	t.Run("module_service_error", func(t *testing.T) {
		mockModuleService := &v1.MockmoduleService{}
		mockLoader := &loader.MockLoader{}

		ctx := context.Background()
		inputRecord := source.Record{
			Key:   []byte("input-key"),
			Value: []byte("input-value"),
			Headers: []source.RecordHeader{
				{Key: "header-key", Value: []byte("header-value")},
			},
			Ctx: ctx,
		}

		mockModuleService.On("Process", mock.Anything, mock.Anything).Return(nil, errors.New("processing error"))
		mockLoader.On("Get", mock.Anything).Return(mockModuleService, nil)

		p := &wasmProcessor{
			ldr: mockLoader,
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := p.process(ctx, inputRecord)
		assert.Error(t, err)
	})
}

func TestWasmProcessor_Close(t *testing.T) {
	outputChannel := make(chan Result)
	p := &wasmProcessor{
		output: outputChannel,
	}

	p.Close()

	_, ok := <-outputChannel
	assert.False(t, ok)
}

func TestNewWasmProcessor(t *testing.T) {
	mockLoader := &loader.MockLoader{}
	inputChannel := make(chan source.Record)

	p, err := NewWasmProcessor(mockLoader, inputChannel, nil)
	assert.NoError(t, err)
	wp, ok := p.(*wasmProcessor)
	assert.True(t, ok)
	assert.Equal(t, mockLoader, wp.ldr)
	assert.EqualValues(t, inputChannel, wp.input)
	assert.NotNil(t, wp.output)
}

func TestWasmProcessor_Reload(t *testing.T) {
	t.Run("reload_signal_triggers_loader_reload", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockLoader := &loader.MockLoader{}
		reloaded := make(chan struct{}, 1)
		mockLoader.On("Reload", mock.Anything).Run(func(args mock.Arguments) {
			reloaded <- struct{}{}
		}).Return(nil)

		in := make(chan source.Record)
		reloadCh := make(chan struct{}, 1)
		p, err := NewWasmProcessor(mockLoader, in, reloadCh)
		require.NoError(t, err)

		go p.Start(ctx)

		reloadCh <- struct{}{}

		select {
		case <-reloaded:
		case <-time.After(2 * time.Second):
			t.Fatal("processor did not reload on signal")
		}
		mockLoader.AssertExpectations(t)
	})

	t.Run("failed_reload_does_not_stop_the_processor", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockLoader := &loader.MockLoader{}
		mockLoader.On("Reload", mock.Anything).Return(errors.New("bad module"))
		// The record below gets consumed and processed asynchronously after
		// this subtest's assertion already succeeded; stub Get (optionally)
		// so that unrelated background processing doesn't panic the mock.
		mockLoader.On("Get", mock.Anything).Return(nil, errors.New("no module")).Maybe()

		in := make(chan source.Record)
		reloadCh := make(chan struct{}, 1)
		p, err := NewWasmProcessor(mockLoader, in, reloadCh)
		require.NoError(t, err)

		go p.Start(ctx)
		reloadCh <- struct{}{}

		// The processor must still be consuming input after a failed reload.
		select {
		case in <- source.Record{Ctx: ctx, Key: []byte("k"), Value: []byte("v")}:
		case <-time.After(2 * time.Second):
			t.Fatal("processor stopped consuming after a failed reload")
		}
	})
}
