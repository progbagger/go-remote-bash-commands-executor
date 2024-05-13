package executor

import (
	"common"
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Struct that allows to run multiple commands in same conditions
type Executor struct {
	Workdir string
	Env     []common.EnvironmentEntry
}

// Runs given command with provided input stream reader and writes its
// output to outWriter and errors to errWriter.
//
// Returns error channel to determine when command is finished and
// cancelation function to interrupt command.
func (executor *Executor) RunScript(
	ctx context.Context,

	inReader io.Reader,
	outWriter io.Writer,
	errWriter io.Writer,

	command string,
	args ...string,
) (<-chan error, func() error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = parseEnv(executor.Env)
	cmd.Dir = executor.Workdir

	cmd.Stdin = inReader
	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	isDone := make(chan error)
	go func() {
		err := cmd.Run()
		isDone <- err
	}()

	return isDone, cmd.Cancel
}

func parseEnv(entries []common.EnvironmentEntry) []string {
	if len(entries) == 0 {
		return nil
	}

	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, fmt.Sprintf("%s=%s", entry.Key, entry.Value))
	}

	return result
}
