package loader

import (
	"context"
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
	l.Loader = l
	return l
}

func (l *s3Loader) Reload(ctx context.Context) error {
	return l.lazyReloader.Reload(ctx)
}

func (l *s3Loader) Resolve(ctx context.Context) {
	// Define a file to download to
	outFile, err := os.Create(l.path)
	if err != nil {
		slog.Error("failed to create file for S3 download", slog.String("err", err.Error()))
		return
	}
	defer outFile.Close()

	// Reload the file
	getObjectOutput, err := l.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &l.bucket,
		Key:    &l.path,
	})
	if err != nil {
		slog.Error("failed to download file from S3", slog.String("err", err.Error()))
		return
	}
	defer getObjectOutput.Body.Close()

	// Write the data to the locally created file
	_, err = io.Copy(outFile, getObjectOutput.Body)
	if err != nil {
		slog.Error("failed to write downloaded file to local disk", slog.String("err", err.Error()))
		return
	}

	slog.Info("Successfully downloaded artifact from bucket",
		slog.String("path", l.path),
		slog.String("bucket", l.bucket),
	)
}
