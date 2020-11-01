package appwatchertools

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"time"
)

type OnFileChange = func(path []string) error

type Watcher = WatchConfig

func NewWatcher() *Watcher {
	return NewWatchConfig()
}

func (w *Watcher) WatchForChanges(ctx context.Context) error {
	if w.Dir == "" {
		return fmt.Errorf("watcher is missing Dir parameter")
	}
	if w.Debounce == 0 {
		w.Debounce = 200 * time.Millisecond
	}
	events := make(chan fsnotify.Event, 100)

	go func() {
		var eventsOut []string
		for {
			select {
			case event := <-events:
				// @todo contemplate errors, delete events via Write?
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					eventsOut = append(eventsOut, event.Name)
				}
			case <-time.After(w.Debounce):
				if w.isRebuilding {
					continue
				}
				if eventsOut != nil {
					w.isRebuilding = true
					if err := w.OnChange(eventsOut); err != nil {
						// @todo some better?
						panic(err)
					}
					eventsOut = nil
					w.isRebuilding = false
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return WatchDir(w, ctx, events)
}
