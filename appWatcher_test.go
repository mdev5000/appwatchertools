package appwatchertools

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
		if isInit {
			logger.Success("initializing")
		}
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
	logger.Success("writing to first")
	writeTmpFile(t, "first", "first file change")

	// write it again
	time.Sleep(1 * time.Second)
	logger.Success("writing to first")
	writeTmpFile(t, "first", "first file change")

	// wait for stuff
	time.Sleep(1 * time.Second)

	// check we got the right messages
	cancel()
	fmt.Println("LOG:")
	for _, m := range logger.messagesTracked {
		fmt.Println(m)
	}
	// @todo eventually write assertion for this.
	//// check events were as expected
	//messages := logger.messagesTracked
	//i := 0
	//require.Equal(t, messages[i], "initializing")
	//i += 1
	//require.Equal(t, messages[i], "on change files: []")
	//i += 1
	//requireRunningAppMsg(t, messages[i])
	//i += 1
	//require.Equal(t, messages[i], "writing to first")
	//i += 1
	//require.Equal(t, messages[i], "on change files: [/Users/matt/devtmp/go/appwatchertools/testdata/tmp/appWatcher_test1/first]")
	//i += 1
	//require.Equal(t, messages[i], "rebuilding")
	//i += 1
	//requireIsStoppingMsg(t, messages[i])
	//i += 1
	//require.Equal(t, messages[i], "on change files: [/Users/matt/devtmp/go/appwatchertools/testdata/tmp/appWatcher_test1/rebuild]")
}

func requireRunningAppMsg(t *testing.T, msg string)  {
	require.True(t, strings.HasPrefix(msg, "Running: /Users/matt/devtmp/go/appwatchertools/testdata/runforevere"))
}

func requireIsStoppingMsg(t *testing.T, msg string)  {
	require.True(t, strings.HasPrefix(msg, "Stopping: PID"))
}
