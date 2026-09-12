package loader

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"log/slog"
	"os"
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
func (l *s3Loader) download(ctx context.Context) error {
	outFile, err := os.Create(l.path)
	if err != nil {
		return fmt.Errorf("failed to create file %s for S3 download: %w", l.path, err)
	}
	defer outFile.Close()

	getObjectOutput, err := l.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &l.bucket,
		Key:    &l.path,
	})
	if err != nil {
		return fmt.Errorf("failed to download %s from bucket %s: %w", l.path, l.bucket, err)
	}
	defer getObjectOutput.Body.Close()

	if _, err = io.Copy(outFile, getObjectOutput.Body); err != nil {
		return fmt.Errorf("failed to write downloaded artifact to %s: %w", l.path, err)
	}

	slog.Info("Successfully downloaded artifact from bucket",
		slog.String("path", l.path),
		slog.String("bucket", l.bucket),
	)

	return nil
}
