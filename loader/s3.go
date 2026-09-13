package loader

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type s3Loader struct {
	lazyReloader
	s3Client S3Client
	bucket   string
}

type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func NewS3Loader(s3Client S3Client, bucket string, pluginRef string) Loader {
	l := &s3Loader{
		s3Client:     s3Client,
		bucket:       bucket,
		lazyReloader: newReloader(pluginRef),
	}
	// Wire the download in as the reloader's resolver. Assigning a function here
	// rather than relying on method promotion is what makes this reachable; see
	// the comment on lazyReloader.resolve.
	l.resolve = l.download
	return l
}

// download fetches the module artifact from S3 to the local path the loader
// reads from. The object key and the local path are both pluginRef.
//
// The fetch and write happen against a temporary file in the same directory
// as l.path, which is only renamed into place once fully written. This keeps
// a failed or partial download from destroying the artifact already on disk:
// if S3 is unreachable, or the copy fails partway through, the existing file
// is left untouched so a process restart still has something to load.
func (l *s3Loader) download(ctx context.Context) error {
	getObjectOutput, err := l.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &l.bucket,
		Key:    &l.path,
	})
	if err != nil {
		return fmt.Errorf("failed to download %s from bucket %s: %w", l.path, l.bucket, err)
	}
	defer getObjectOutput.Body.Close()

	tmpFile, err := os.CreateTemp(filepath.Dir(l.path), "."+filepath.Base(l.path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create file %s for S3 download: %w", l.path, err)
	}
	tmpPath := tmpFile.Name()
	renamed := false
	defer func() {
		if !renamed {
			os.Remove(tmpPath)
		}
	}()

	if _, err = io.Copy(tmpFile, getObjectOutput.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write downloaded artifact to %s: %w", l.path, err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to write downloaded artifact to %s: %w", l.path, err)
	}

	if err = os.Rename(tmpPath, l.path); err != nil {
		return fmt.Errorf("failed to write downloaded artifact to %s: %w", l.path, err)
	}
	renamed = true

	slog.Info("Successfully downloaded artifact from bucket",
		slog.String("path", l.path),
		slog.String("bucket", l.bucket),
	)

	return nil
}
