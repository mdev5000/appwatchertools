package main

import (
	"github.com/mdev5000/appwatchertools"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ensureOk(err error) {
	if err != nil {
		panic(err)
	}
}

type runner struct {
	appWatcher *appwatchertools.AppWatcher
}

func main() {
	r := &runner{
		appWatcher: appwatchertools.NewAppWatcher(),
	}

	// Set the watched directory to the application working directory.
	wd, err := os.Getwd()
	ensureOk(err)
	r.appWatcher.Dir = wd

	// Only watch application files.
	r.appWatcher.FileFilter = func(path string) (bool, error) {
		return r.isApplicationPath(wd, path), nil
	}

	r.appWatcher.OnChangeFn = r.onChange

	// Specify application to restart when a file has changed.
	r.appWatcher.CommandFn = func() *exec.Cmd {
		return exec.Command("./app")
	}

	// Run the watcher.
	ctx := context.Background()
	ensureOk(r.appWatcher.Run(ctx))
}

// When a file changes run build and main applications and is there's no errors
// start the application.
func (r *runner) onChange(files []string, isInit bool) bool {
	// Do any pre-building here.
	// You can also vary what is run based on what files have changed.
	// isInit indicate the watcher is booting up and things should probably run regardless of changed files.

	// Then compile the application so it can be run.
	if err := r.appWatcher.RunCommand("go", "build", "-o", "app", "main/main.go"); err != nil {
		r.appWatcher.ExeLogger.Error(err)
		return false
	}
	return true
}

// Limit what files and directories are watched:
func (r *runner) isApplicationPath(dir string, path string) bool {
	return strings.HasPrefix(path, filepath.Join(dir, "example", "watcher.go"))
}
