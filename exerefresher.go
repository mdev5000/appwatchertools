package appwatchertools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type ExeLogger interface {
	Success(msg string, args ...interface{})
	Error(error)
}

type MakeCommandFn = func() *exec.Cmd

type ExeRefresher struct {
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	//ID        string
	Logger    ExeLogger
	Restart   chan bool
	CommandFn MakeCommandFn
	//Debug      bool
	//cancelFunc context.CancelFunc
	//context    context.Context
	//gil        *sync.Once
}

func NewExeRefresher() *ExeRefresher {
	return &ExeRefresher{
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Restart:   make(chan bool),
		Logger:    nil,
		CommandFn: nil,
	}
}

func (r *ExeRefresher) Run(ctx context.Context) {
	var cmd *exec.Cmd
	for {
		select {
		case <- ctx.Done():
			if cmd != nil && cmd.Process != nil {
				cmd.Process.Kill()
				return
			}

		case <-r.Restart:
			if cmd != nil {
				// kill the previous command
				if cmd.Process != nil {
					pid := cmd.Process.Pid
					r.Logger.Success("Stopping: PID %d", pid)
					if err := cmd.Process.Kill(); err != nil {
						r.Logger.Error(err)
					}
				}
			}
			//if r.Debug {
			//	bp := r.FullBuildPath()
			//	args := []string{"exec", bp}
			//	args = append(args, r.CommandFlags...)
			//	cmd = exec.Command("dlv", args...)
			//} else {
			//	cmd = exec.Command(r.FullBuildPath(), r.CommandFlags...)
			//}
			cmd = r.CommandFn()
			go func() {
				err := r.runAndListen(cmd)
				if err != nil {
					r.Logger.Error(err)
				}
			}()
		}
	}
}

func (r *ExeRefresher) runAndListen(cmd *exec.Cmd) error {
	cmd.Stderr = r.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

	cmd.Stdin = r.Stdin
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = r.Stdout
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}

	var stderr bytes.Buffer

	cmd.Stderr = io.MultiWriter(&stderr, cmd.Stderr)

	//// Set the environment variables from config
	//if len(r.CommandEnv) != 0 {
	//	cmd.Env = append(r.CommandEnv, os.Environ()...)
	//}

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("%s\n%s", err, stderr.String())
	}

	r.Logger.Success("Running: %s (PID: %d)", strings.Join(cmd.Args, " "), cmd.Process.Pid)
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("%s\n%s", err, stderr.String())
	}
	return nil
}
