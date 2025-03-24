package loader

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"log"
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
		log.Fatalf("failed to create file for S3 download: %v", err)
	}
	defer outFile.Close()

	// Reload the file
	getObjectOutput, err := l.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &l.bucket,
		Key:    &l.path,
	})
	if err != nil {
		log.Fatalf("failed to download file from S3: %v", err)
	}
	defer getObjectOutput.Body.Close()

	// Write the data to the locally created file
	_, err = io.Copy(outFile, getObjectOutput.Body)
	if err != nil {
		log.Fatalf("failed to write downloaded file to local disk: %v", err)
	}

	log.Printf("Successfully downloaded %s from bucket %s to %s", l.path, l.bucket, l.path)
}
