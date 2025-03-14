package sink

import "context"

type Sink interface {
	Write(ctx context.Context)
	Close()
}
