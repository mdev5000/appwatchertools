package appwatchertools

import (
	"context"
	"errors"
	"github.com/fsnotify/fsnotify"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WatchFileFilter = func(path string) (bool, error)
type OnChangeFn = func(path []string) error

type WatchConfig struct {
	Dir    string
	Debounce time.Duration
	Filter WatchFileFilter
	OnChange OnChangeFn
}

func NewWatchConfig() *WatchConfig {
	return &WatchConfig{
		Filter: func(path string) (bool, error) {
			return true, nil
		},
	}
}

func (c *WatchConfig) cancelFunc() {
	// @todo create this
}

func WatchDir(cfg *WatchConfig, ctx context.Context, events chan<- fsnotify.Event) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				events <- event
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	keepWatching(ctx, watcher, cfg)
	<-ctx.Done()
	return nil
}

func keepWatching(ctx context.Context, watcher *fsnotify.Watcher, cfg *WatchConfig) {
	go func() {
		for {
			err := filepath.Walk(cfg.Dir, func(path string, info os.FileInfo, err error) error {
				if info == nil {
					cfg.cancelFunc()
					return errors.New("nil directory")
				}
				if info.IsDir() {
					if strings.HasPrefix(filepath.Base(path), "_") {
						return filepath.SkipDir
					}
					if len(path) > 1 && strings.HasPrefix(filepath.Base(path), ".") {
						return filepath.SkipDir
					}
				}
				shouldWatchFile, err := cfg.Filter(path)
				if shouldWatchFile {
					err = watcher.Add(path)
				}
				return err
			})

			if err != nil {
				ctx.Done()
				break
			}
			// sweep for new files every 1 second
			time.Sleep(1 * time.Second)
		}
	}()
}
