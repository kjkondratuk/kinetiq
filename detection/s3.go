package detection

import "time"

type S3EventNotification struct {
	Records []S3EventRecord `json:"Records"`
}

type S3EventRecord struct {
	EventVersion string    `json:"eventVersion"`
	EventSource  string    `json:"eventSource"`
	AWSRegion    string    `json:"awsRegion"`
	EventTime    time.Time `json:"eventTime"`
	EventName    string    `json:"eventName"`
	S3           S3Entity  `json:"s3"`
}

type S3Entity struct {
	Bucket S3BucketEntity `json:"bucket"`
	Object S3ObjectEntity `json:"object"`
}

type S3BucketEntity struct {
	Name string `json:"name"`
	Arn  string `json:"arn"`
}

type S3ObjectEntity struct {
	Key       string `json:"key"`
	Size      int64  `json:"size"`
	ETag      string `json:"eTag"`
	VersionId string `json:"versionId,omitempty"`
}
