package appwatchertools

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

var TmpDir string
var FixtureDir string

func init() {
	wd, _ := os.Getwd()
	TmpDir = filepath.Join(wd, "testdata", "tmp", "appWatcher_test1")
	FixtureDir = filepath.Join(wd, "testdata")
}

type testLogger struct {
	messages chan string
	messagesTracked []string
}

func newTestLogger() *testLogger {
	return &testLogger{make(chan string, 100), nil}
}

func (t *testLogger) Success(msg string, args ...interface{}) {
	t.messages <- fmt.Sprintf(msg, args...)
}

func (t *testLogger) Error(err error) {
	t.messages <- err.Error()
}

func (t *testLogger) TrackMessages(ctx context.Context) {
	go func() {
		for {
			select {
			case <- ctx.Done():
				return
			case msg := <- t.messages:
				t.messagesTracked = append(t.messagesTracked, msg)
			}
		}
	}()
}

func writeTmpFile(t *testing.T, relativePath, contents string) {
	require.Nil(t, ioutil.WriteFile(filepath.Join(TmpDir, relativePath), []byte(contents), 0776))
}

func TestHandlesFilesChangingDuringRebuild(t *testing.T) {
	require.Nil(t, os.MkdirAll(TmpDir, 0776))
	aw := NewAppWatcher()
	aw.Dir = TmpDir
	logger := newTestLogger()
	aw.ExeLogger = logger
	aw.Debounce = 200 * time.Millisecond
	aw.FileFilter = func(path string) (bool, error) {
		return true, nil
	}
	aw.OnChangeFn = func(fileChanges []string, isInit bool) bool {
		logger.Success("on change files: %s", fileChanges)
		for _, file := range fileChanges {
			if file == filepath.Join(TmpDir, "first") {
				logger.Success("rebuilding")
				writeTmpFile(t, "rebuild", "file change on rebuild")
			}
		}
		return true
	}
	aw.CommandFn = func() *exec.Cmd {
		return exec.Command(filepath.Join(FixtureDir, "runforevere"))
	}
	ctx, cancel := context.WithCancel(context.Background())

	logger.TrackMessages(ctx)
	go func() {
		require.Nil(t, aw.Run(ctx))
	}()

	// start writing file changes
	time.Sleep(1 * time.Second)
	time.Sleep(100 * time.Millisecond)
	fmt.Println("write first")
	writeTmpFile(t, "first", "first file change")

	// write it again
	time.Sleep(100 * time.Millisecond)
	fmt.Println("write first")
	writeTmpFile(t, "first", "first file change")

	// wait for stuff
	time.Sleep(1 * time.Second)

	// check we got the right messages
	cancel()
	fmt.Println("LOG:")
	for _, m := range logger.messagesTracked {
		fmt.Println(m)
	}
}