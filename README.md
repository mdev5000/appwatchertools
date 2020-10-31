# appwatchertools

Utility functions for setting up an application watcher.

## Get started

```bash
go get bitbucket.org/mdev5000/appwatchertools
```

## Example

`watcher/watcher.go`
```go
package main

import (
	"bitbucket.org/mdev5000/appwatchertools"
	"context"
	"fmt"
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

	wd, err := os.Getwd()
	ensureOk(err)
	r.appWatcher.Dir = wd

	// Only watch application files.
	r.appWatcher.FileFilter = func(path string) (bool, error) {
		return r.isApplicationPath(wd, path), nil
	}

	r.appWatcher.OnChangeFn = r.onChange

	// OnChange will rebuild the application and remake the exe ./app, after that
	// we can run the application.
	r.appWatcher.CommandFn = func() *exec.Cmd {
		// Usually you would run the compiled application here.
		// return exec.Command("./app")
		return exec.Command("echo", "running app")
	}

	// Run the watcher.
	ctx := context.Background()
	ensureOk(r.appWatcher.Run(ctx))
}

// When a file changes run build and main applications and is there's no errors
// start the application.
func (r *runner) onChange(files []string, isInit bool) bool {
	fmt.Println("building things")

	// Usually you could build the app here:
	//
	//if err := r.appWatcher.RunCommand("go", "build", "-o", "app", "main/main.go"); err != nil {
	//	r.appWatcher.ExeLogger.Error(err)
	//	return false
	//}

	if err := r.appWatcher.RunCommand("echo", "building"); err != nil {
		r.appWatcher.ExeLogger.Error(err)
		return false
	}
	return true
}

// Limit what files and directories are watched
// In this case anything in:
//	[rootdir]/main/
//	[rootdir]/app/
func (r *runner) isApplicationPath(dir string, path string) bool {
	return strings.HasPrefix(path, filepath.Join(dir, "main")) ||
		strings.HasPrefix(path, filepath.Join(dir, "app"))
}
```

You can then run the watcher with:

```bash
go run watcher/watcher.go
```