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
	"path/filepath"
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
	assert.NotNil(t, s3Loader.resolve, "Resolver must be wired so Reload and Get can fetch the artifact")
}

// TestS3Loader_Reload_DownloadsArtifact is the regression test for the bug where
// s3Loader.Resolve was unreachable. lazyReloader declared its own Resolve method,
// which shadowed the one on s3Loader, so Reload closed the runtime and silently
// reloaded the stale local file instead of fetching the new object.
func TestS3Loader_Reload_DownloadsArtifact(t *testing.T) {
	// Arrange
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test-plugin.wasm")

	mockS3Client := NewMockS3Client(t)
	mockS3Client.EXPECT().GetObject(mock.Anything, mock.MatchedBy(func(params *s3.GetObjectInput) bool {
		return *params.Bucket == "test-bucket" && *params.Key == path
	}), mock.Anything).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader([]byte("new module bytes"))),
	}, nil).Once()

	mockPluginLoader := &MockpluginLoader{}
	mockPluginLoader.On("load", mock.Anything, mock.Anything, path).Return(&MockcloseablePlugin{}, nil)

	loader := NewS3Loader(mockS3Client, "test-bucket", path).(*s3Loader)
	loader.pluginLoader = mockPluginLoader

	// Act
	err := loader.Reload(ctx)

	// Assert
	assert.NoError(t, err, "Expected no error from Reload")
	mockS3Client.AssertExpectations(t) // fails if the download never happened
	mockPluginLoader.AssertExpectations(t)

	content, err := os.ReadFile(path)
	assert.NoError(t, err, "Artifact should have been written to the local path")
	assert.Equal(t, "new module bytes", string(content), "Local artifact should hold the downloaded bytes")
}

// A failed download must leave the currently loaded module running rather than
// tearing it down and leaving the process unable to process anything.
func TestS3Loader_Reload_AbortsBeforeClosingOnDownloadFailure(t *testing.T) {
	// Arrange
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test-plugin.wasm")

	mockS3Client := NewMockS3Client(t)
	mockS3Client.EXPECT().GetObject(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("s3 unavailable"))

	// No Close or load expectations: the running plugin must be left alone.
	runningPlugin := &MockcloseablePlugin{}
	mockPluginLoader := &MockpluginLoader{}

	loader := NewS3Loader(mockS3Client, "test-bucket", path).(*s3Loader)
	loader.pluginLoader = mockPluginLoader
	loader.closeablePlugin = runningPlugin

	// Act
	err := loader.Reload(ctx)

	// Assert
	assert.Error(t, err, "A failed download must surface as an error")
	assert.Contains(t, err.Error(), "failed to resolve plugin artifact for reload")
	runningPlugin.AssertNotCalled(t, "Close", mock.Anything)
	mockPluginLoader.AssertNotCalled(t, "load", mock.Anything, mock.Anything, mock.Anything)
	assert.Equal(t, runningPlugin, loader.closeablePlugin, "The loaded module should be untouched")
}

// A cold start has nothing on local disk, so the first Get must resolve too.
func TestS3Loader_Get_ResolvesOnColdStart(t *testing.T) {
	// Arrange
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test-plugin.wasm")

	mockS3Client := NewMockS3Client(t)
	mockS3Client.EXPECT().GetObject(mock.Anything, mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader([]byte("initial module bytes"))),
	}, nil).Once()

	mockPluginLoader := &MockpluginLoader{}
	mockPluginLoader.On("load", mock.Anything, mock.Anything, path).Return(&MockcloseablePlugin{}, nil)

	loader := NewS3Loader(mockS3Client, "test-bucket", path).(*s3Loader)
	loader.pluginLoader = mockPluginLoader

	// Act
	plugin, err := loader.Get(ctx)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, plugin)
	mockS3Client.AssertExpectations(t)

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "initial module bytes", string(content))
}

func TestS3Loader_download(t *testing.T) {
	t.Run("successful_download", func(t *testing.T) {
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "test-plugin.wasm")

		mockS3Client := NewMockS3Client(t)
		mockS3Client.EXPECT().GetObject(mock.Anything, mock.MatchedBy(func(params *s3.GetObjectInput) bool {
			return *params.Bucket == "test-bucket" && *params.Key == path
		}), mock.Anything).Return(&s3.GetObjectOutput{
			Body: io.NopCloser(bytes.NewReader([]byte("test content"))),
		}, nil)

		loader := NewS3Loader(mockS3Client, "test-bucket", path).(*s3Loader)

		err := loader.download(ctx)

		assert.NoError(t, err)
		content, err := os.ReadFile(path)
		assert.NoError(t, err, "Failed to read downloaded file")
		assert.Equal(t, "test content", string(content), "File content doesn't match expected")
	})

	t.Run("file_creation_error", func(t *testing.T) {
		ctx := context.Background()
		mockS3Client := NewMockS3Client(t)

		loader := NewS3Loader(mockS3Client, "test-bucket", "/nonexistent/directory/test-plugin").(*s3Loader)

		err := loader.download(ctx)

		assert.Error(t, err, "A file creation failure must be returned, not just logged")
		assert.Contains(t, err.Error(), "failed to create file")
	})

	t.Run("s3_get_object_error", func(t *testing.T) {
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "test-plugin.wasm")

		mockS3Client := NewMockS3Client(t)
		mockS3Client.EXPECT().GetObject(mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("s3 error"))

		loader := NewS3Loader(mockS3Client, "test-bucket", path).(*s3Loader)

		err := loader.download(ctx)

		assert.Error(t, err, "A download failure must be returned, not just logged")
		assert.Contains(t, err.Error(), "failed to download")
	})

	t.Run("io_copy_error", func(t *testing.T) {
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "test-plugin.wasm")

		mockS3Client := NewMockS3Client(t)
		mockS3Client.EXPECT().GetObject(mock.Anything, mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{
			Body: io.NopCloser(&errorReader{err: errors.New("read error")}),
		}, nil)

		loader := NewS3Loader(mockS3Client, "test-bucket", path).(*s3Loader)

		err := loader.download(ctx)

		assert.Error(t, err, "A copy failure must be returned, not just logged")
		assert.Contains(t, err.Error(), "failed to write downloaded artifact")
	})
}

// errorReader is a mock io.Reader that always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}
