package loader

import (
	"bytes"
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"os"
	"testing"
)

func TestNewS3Loader(t *testing.T) {
	// Arrange
	mockS3Client := NewMockS3Client(t)
	bucket := "test-bucket"
	pluginRef := "test-plugin"

	// Act
	loader := NewS3Loader(mockS3Client, bucket, pluginRef)

	// Assert
	s3Loader, ok := loader.(*s3Loader)
	assert.True(t, ok, "Expected loader to be of type *s3Loader")
	assert.Equal(t, mockS3Client, s3Loader.s3Client, "S3 client not set correctly")
	assert.Equal(t, bucket, s3Loader.bucket, "Bucket not set correctly")
	assert.Equal(t, pluginRef, s3Loader.path, "Plugin reference not set correctly")
	assert.Equal(t, s3Loader, s3Loader.Loader, "Loader not set correctly")
}

func TestS3Loader_Reload(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockS3Client := NewMockS3Client(t)
	mockPluginLoader := &MockpluginLoader{}

	// Create a mock plugin for the loader to return
	mockPlugin := &MockcloseablePlugin{}
	mockPluginLoader.On("load", mock.Anything, mock.Anything, "test-plugin").Return(mockPlugin, nil)

	loader := &s3Loader{
		s3Client: mockS3Client,
		bucket:   "test-bucket",
		lazyReloader: lazyReloader{
			path:         "test-plugin",
			pluginLoader: mockPluginLoader,
		},
	}
	loader.Loader = loader

	// Act
	err := loader.Reload(ctx)

	// Assert
	assert.NoError(t, err, "Expected no error from Reload")
	mockPluginLoader.AssertExpectations(t)
}

func TestS3Loader_Resolve(t *testing.T) {
	t.Run("successful_download", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		mockS3Client := NewMockS3Client(t)
		mockPluginLoader := &MockpluginLoader{}
		loader := &s3Loader{
			s3Client: mockS3Client,
			bucket:   "test-bucket",
			lazyReloader: lazyReloader{
				path:         "test-plugin",
				pluginLoader: mockPluginLoader,
			},
		}

		// Create a temporary file that will be used by the Resolve method
		tmpFile, err := os.CreateTemp("", "test-plugin")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Override the path to use our temp file
		loader.path = tmpFile.Name()

		// Mock the GetObject method to return a successful response
		mockBody := io.NopCloser(bytes.NewReader([]byte("test content")))
		mockOutput := &s3.GetObjectOutput{
			Body: mockBody,
		}
		mockS3Client.EXPECT().GetObject(mock.Anything, mock.MatchedBy(func(params *s3.GetObjectInput) bool {
			return *params.Bucket == "test-bucket" && *params.Key == tmpFile.Name()
		}), mock.Anything).Return(mockOutput, nil)

		// Act
		loader.Resolve(ctx)

		// Assert
		// Verify the file was written correctly
		content, err := os.ReadFile(tmpFile.Name())
		assert.NoError(t, err, "Failed to read temp file")
		assert.Equal(t, "test content", string(content), "File content doesn't match expected")
	})

	t.Run("file_creation_error", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		mockS3Client := NewMockS3Client(t)
		mockPluginLoader := &MockpluginLoader{}
		loader := &s3Loader{
			s3Client: mockS3Client,
			bucket:   "test-bucket",
			lazyReloader: lazyReloader{
				path:         "/nonexistent/directory/test-plugin", // This should cause an error when trying to create the file
				pluginLoader: mockPluginLoader,
			},
		}

		// Act
		loader.Resolve(ctx)

		// Assert
		// No assertions needed as we're just verifying it doesn't panic
		// The error is logged but not returned
	})

	t.Run("s3_get_object_error", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		mockS3Client := NewMockS3Client(t)
		mockPluginLoader := &MockpluginLoader{}
		loader := &s3Loader{
			s3Client: mockS3Client,
			bucket:   "test-bucket",
			lazyReloader: lazyReloader{
				path:         "test-plugin",
				pluginLoader: mockPluginLoader,
			},
		}

		// Create a temporary file that will be used by the Resolve method
		tmpFile, err := os.CreateTemp("", "test-plugin")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Override the path to use our temp file
		loader.path = tmpFile.Name()

		// Mock the GetObject method to return an error
		mockS3Client.EXPECT().GetObject(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("s3 error"))

		// Act
		loader.Resolve(ctx)

		// Assert
		// No assertions needed as we're just verifying it doesn't panic
		// The error is logged but not returned
	})

	t.Run("io_copy_error", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		mockS3Client := NewMockS3Client(t)
		mockPluginLoader := &MockpluginLoader{}
		loader := &s3Loader{
			s3Client: mockS3Client,
			bucket:   "test-bucket",
			lazyReloader: lazyReloader{
				path:         "test-plugin",
				pluginLoader: mockPluginLoader,
			},
		}

		// Create a temporary file that will be used by the Resolve method
		tmpFile, err := os.CreateTemp("", "test-plugin")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Override the path to use our temp file
		loader.path = tmpFile.Name()

		// Mock the GetObject method to return a reader that will cause an error
		errReader := &errorReader{err: errors.New("read error")}
		mockBody := io.NopCloser(errReader)
		mockOutput := &s3.GetObjectOutput{
			Body: mockBody,
		}
		mockS3Client.EXPECT().GetObject(mock.Anything, mock.Anything, mock.Anything).Return(mockOutput, nil)

		// Act
		loader.Resolve(ctx)

		// Assert
		// No assertions needed as we're just verifying it doesn't panic
		// The error is logged but not returned
	})
}

// errorReader is a mock io.Reader that always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}
