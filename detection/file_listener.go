package detection

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"log"
	"strings"
)

type fileListener struct {
	watcher *fsnotify.Watcher
	path    *string
}

type FileWatcherNotification struct {
	Path string
}

func NewFilesystemListener(watcher *fsnotify.Watcher) Listener[FileWatcherNotification] {
	return &fileListener{watcher: watcher}
}

func NewPathWatcher(path string) (Listener[FileWatcherNotification], error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create path watcher: %w", err)
	}

	parts := strings.Split(path, "/")
	parentDir := strings.Join(parts[0:len(parts)-1], "/")

	err = watcher.Add(parentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to add path to watcher: %w", err)
	}

	w := &fileListener{watcher: watcher}

	w.path = &path

	return w, nil
}

func (f *fileListener) Listen(responder Responder[FileWatcherNotification]) {
	for {
		select {
		case event, ok := <-f.watcher.Events:
			log.Printf("event: %s", event)
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				// if this is a file watcher, make sure the file matches, otherwise send all write/create events
				if f.path != nil {
					log.Printf("path: %s - name: %s", *f.path, event.Name)
					if event.Name == *f.path {
						responder(FileWatcherNotification{Path: event.Name})
					}
				} else {
					responder(FileWatcherNotification{Path: event.Name})
				}
			}
		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("error watching file: err: %s", err)
		}
	}
}

func (f *fileListener) Close() {
	err := f.watcher.Close()
	if err != nil {
		fmt.Printf("failed to close file watcher: %s", err)
	}
}
