package appwatchertools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type DefaultLogger struct {
}

func (e DefaultLogger) Success(msg string, args ...interface{}) {
	fmt.Printf(msg + "\n", args...)
}

func (e DefaultLogger) Error(err error) {
	fmt.Println(err.Error())
}

var _ ExeLogger = DefaultLogger{}

type OnChange = func(fileChanges []string, isInit bool) bool

type restartInfo struct {
	Force        bool
	ChangedFiles []string
}

type AppWatcher struct {
	Dir        string
	ExeLogger  ExeLogger
	FileFilter WatchFileFilter
	OnChangeFn OnChange
	CommandFn  MakeCommandFn
}

func NewAppWatcher() *AppWatcher {
	return &AppWatcher{ExeLogger: DefaultLogger{}}
}

func (a *AppWatcher) Run(ctx context.Context) error {
	if err := a.validateWatcher(); err != nil {
		return err
	}
	exeRefesher := NewExeRefresher()
	exeRefesher.Logger = a.ExeLogger
	exeRefesher.CommandFn = a.CommandFn

	restart := make(chan restartInfo, 10)
	go exeRefesher.Run(ctx)
	go a.runAppWatcher(ctx, exeRefesher.Restart, restart)
	w := NewWatcher()
	w.Dir = a.Dir
	w.Filter = a.FileFilter
	w.OnChange = func(paths []string) error {
		restart <- restartInfo{ChangedFiles: paths}
		return nil
	}
	restart <- restartInfo{Force: true}
	return w.WatchForChanges(ctx)
}

func (a *AppWatcher) validateWatcher() error {
	if a.Dir == "" {
		return fmt.Errorf("must specify the Dir parameter for the AppWatcher")
	}
	if a.ExeLogger == nil {
		return fmt.Errorf("must specify the ExeLogger parameter for the AppWatcher")
	}
	if a.FileFilter == nil {
		return fmt.Errorf("must specify the FileFilter parameter for the AppWatcher")
	}
	if a.OnChangeFn == nil {
		return fmt.Errorf("must specify the OnChangeFn parameter for the AppWatcher")
	}
	if a.CommandFn == nil {
		return fmt.Errorf("must specify the CommandFn parameter for the AppWatcher")
	}
	return nil
}

func (a *AppWatcher) runAppWatcher(ctx context.Context, restartExe chan<- bool, recompile <-chan restartInfo) {
	for {
		select {
		case <-ctx.Done():
			return
		case recompileInfo := <-recompile:
			shouldRestartExe := a.OnChangeFn(recompileInfo.ChangedFiles, recompileInfo.Force)
			if shouldRestartExe {
				restartExe <- true
			}
		}
	}
}

// Simple utility function for running a command.
func (a *AppWatcher) RunCommand(cmd string, args ...string) error {
	pre := exec.Command(cmd, args...)
	pre.Stdin = os.Stdin
	pre.Stdout = os.Stdout
	pre.Stderr = os.Stderr
	if err := pre.Start(); err != nil {
		return err
	}
	if err := pre.Wait(); err != nil {
		return err
	}
	return nil
}
