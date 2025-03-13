package source

import "context"

type Source interface {
	Read(ctx context.Context)
	Output() <-chan Record
	Enable()
	Disable()
	Close()
}
