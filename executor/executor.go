package executor

import (
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Struct that represents environment variable.
// Will be parsed by executor in form of "Key"="Value".
type EnvironmentEntry struct {
	Key string `json:"key"`
	Val string `json:"value"`
}

// Method for postgresql to able to push such values into the database
func (enrty EnvironmentEntry) Value() (driver.Value, error) {
	return fmt.Sprintf("(%s,%s)", enrty.Key, enrty.Val), nil
}

func (entry *EnvironmentEntry) Scan(value any) error {
	if b, ok := value.([]byte); ok {
		splitted := strings.Split(strings.Trim(string(b), "()"), ",")
		if len(splitted) != 2 {
			return fmt.Errorf("unknown environment entry")
		}

		entry.Key = splitted[0]
		entry.Val = splitted[1]
	} else {
		entry.Key = ""
		entry.Val = ""
	}

	return nil
}

// Struct that allows to run multiple commands in same conditions
type Executor struct {
	Workdir string
	Env     []EnvironmentEntry
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
) <-chan error {
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
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
		result = append(result, fmt.Sprintf("%s=%s", entry.Key, entry.Val))
	}

	return result
}
