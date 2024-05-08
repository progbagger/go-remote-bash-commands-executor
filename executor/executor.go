package executor

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

type EnvironmentEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Executor struct {
	Workdir string
	Env     []EnvironmentEntry
}

func (executor *Executor) RunScript(
	ctx context.Context,

	inReader io.Reader,
	outWriter io.Writer,
	errWriter io.Writer,

	command string,
	args ...string,
) <-chan error {
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

	return isDone
}

func parseEnv(entries []EnvironmentEntry) []string {
	if len(entries) == 0 {
		return nil
	}

	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, fmt.Sprintf("%s=%s", entry.Key, entry.Value))
	}

	return result
}
