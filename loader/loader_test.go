package loader

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLazyReloader_Get(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		pluginLoader func() *MockpluginLoader
		wantErr      bool
	}{
		{
			name: "success",
			path: "valid/path",
			pluginLoader: func() *MockpluginLoader {
				mockPlugin := new(MockcloseablePlugin)
				mockPluginLoader := new(MockpluginLoader)
				mockPluginLoader.On("load", mock.Anything, mock.Anything, mock.Anything).Return(mockPlugin, nil)
				return mockPluginLoader
			},
			wantErr: false,
		},
		{
			name: "error_load",
			path: "invalid/path",
			pluginLoader: func() *MockpluginLoader {
				mockPlugin := new(MockcloseablePlugin)
				mockPluginLoader := new(MockpluginLoader)
				mockPluginLoader.On("load", mock.Anything, mock.Anything, mock.Anything).Return(mockPlugin, fmt.Errorf("load error"))
				return mockPluginLoader
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reloader := &lazyReloader{
				pluginLoader: tt.pluginLoader(),
				path:         tt.path,
			}

			ctx := context.TODO()
			moduleSvc, err := reloader.Get(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, moduleSvc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, moduleSvc)
			}
		})
	}
}

func TestLazyReloader_Reload(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		pluginLoader func() *MockpluginLoader
		//closeableMock func() *MockcloseablePlugin
		wantErr bool
	}{
		{
			name: "success",
			path: "valid/path",
			pluginLoader: func() *MockpluginLoader {
				mockPlugin := new(MockcloseablePlugin)
				mockPluginLoader := new(MockpluginLoader)
				mockPluginLoader.On("load", mock.Anything, mock.Anything, mock.Anything).Return(mockPlugin, nil)
				return mockPluginLoader
			},
			//closeableMock: func() *MockcloseablePlugin {
			//	mockPlugin := new(MockcloseablePlugin)
			//	mockPlugin.On("Close", mock.Anything).Return(nil)
			//	return mockPlugin
			//},
			wantErr: false,
		},
		{
			name: "error_close",
			path: "valid/path",
			pluginLoader: func() *MockpluginLoader {
				mockPluginLoader := new(MockpluginLoader)
				mockPluginLoader.On("load", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("load error"))
				return mockPluginLoader
			},
			//closeableMock: func() *MockcloseablePlugin {
			//	mockPlugin := new(MockcloseablePlugin)
			//	mockPlugin.On("Close", mock.Anything).Return(fmt.Errorf("close error"))
			//	return mockPlugin
			//},
			wantErr: true,
		},
		{
			name: "error_load",
			path: "invalid/path",
			pluginLoader: func() *MockpluginLoader {
				mockPluginLoader := new(MockpluginLoader)
				mockPluginLoader.On("load", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("load error"))
				return mockPluginLoader
			},
			//closeableMock: func() *MockcloseablePlugin {
			//	mockPlugin := new(MockcloseablePlugin)
			//	mockPlugin.On("Close", mock.Anything).Return(nil)
			//	return mockPlugin
			//},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reloader := &lazyReloader{
				pluginLoader: tt.pluginLoader(),
				path:         tt.path,
			}

			ctx := context.TODO()
			err := reloader.Reload(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLazyReloader_Close(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(mockCloseable *MockcloseablePlugin)
		wantErr bool
	}{
		{
			name: "success_close",
			setup: func(mockCloseable *MockcloseablePlugin) {
				mockCloseable.On("Close", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error_close",
			setup: func(mockCloseable *MockcloseablePlugin) {
				mockCloseable.On("Close", mock.Anything).Return(fmt.Errorf("close error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCloseable := new(MockcloseablePlugin)
			tt.setup(mockCloseable)

			reloader := &lazyReloader{
				closeablePlugin: mockCloseable,
			}
			ctx := context.TODO()

			err := reloader.Close(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
