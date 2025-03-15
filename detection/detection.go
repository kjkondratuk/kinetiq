package detection

type Detectable interface {
	S3EventNotification | FileWatcherNotification
}

type Listener[T Detectable] interface {
	Listen(responder Responder[T])
	Close()
}

type Responder[T Detectable] func(notification T)
