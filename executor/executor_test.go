package executor

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunScriptSimpleCommand(t *testing.T) {
	executor := Executor{}

	out := bytes.Buffer{}
	errs := bytes.Buffer{}

	isDone := executor.RunScript(context.Background(), nil, &out, &errs, "echo -n amogus")
	err := <-isDone

	if errs.String() != "" {
		t.Fatalf("errors must be empty, got \"%s\"", errs.String())
	}
	if out.String() != "amogus" {
		t.Fatalf("\"echo amogus\" must print \"amogus\", got \"%s\"", out.String())
	}
	if err != nil {
		t.Fatalf("runner had to return nil, but returned \"%s\"", err)
	}
}

func TestRunScriptSimpleCommandFailed(t *testing.T) {
	executor := Executor{}

	out := bytes.Buffer{}
	errs := bytes.Buffer{}

	isDone := executor.RunScript(context.Background(), nil, &out, &errs, "sus")
	err := <-isDone

	if errs.String() == "" {
		t.Fatalf("errors must not be empty")
	}
	if out.String() != "" {
		t.Fatalf("output must be empty")
	}
	if exitErr, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("runner had to return ExitError, but got %T", err)
	} else if exitErr.ExitCode() == 0 {
		t.Fatalf("exit code must not be 0")
	}
}

func TestRunScriptInterrupted(t *testing.T) {
	executor := Executor{}

	out := bytes.Buffer{}
	errs := bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()

	isDone := executor.RunScript(ctx, nil, &out, &errs, "sleep 10")
	err := <-isDone

	if errs.String() != "" {
		t.Fatalf("errors must be empty")
	}
	if out.String() != "" {
		t.Fatalf("output must be empty")
	}
	if exitErr, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("runner had to return ExitError, but got %T", err)
	} else if exitErr.ExitCode() != -1 {
		t.Fatalf("exit code must be -1, got %d", exitErr.ExitCode())
	}
}

func TestRunScriptWorkdir(t *testing.T) {
	executor := Executor{Workdir: "/"}

	out := bytes.Buffer{}
	errs := bytes.Buffer{}

	isDone := executor.RunScript(context.Background(), nil, &out, &errs, "pwd")
	err := <-isDone

	if errs.String() != "" {
		t.Fatalf("errors must be empty, got \"%s\"", errs.String())
	}
	if out.String() != "/\n" {
		t.Fatalf("\"pwd\" must print \"/\", got \"%s\"", out.String())
	}
	if err != nil {
		t.Fatalf("runner had to return nil, but returned \"%s\"", err)
	}

	executor.Workdir = "/var"

	out = bytes.Buffer{}
	errs = bytes.Buffer{}

	isDone = executor.RunScript(context.Background(), nil, &out, &errs, "pwd")
	err = <-isDone

	if errs.String() != "" {
		t.Fatalf("errors must be empty, got \"%s\"", errs.String())
	}
	if out.String() != "/var\n" {
		t.Fatalf("\"pwd\" must print \"/var\", got \"%s\"", out.String())
	}
	if err != nil {
		t.Fatalf("runner had to return nil, but returned \"%s\"", err)
	}
}

func TestRunScriptEnv(t *testing.T) {
	executor := Executor{Env: []EnvironmentEntry{{Key: "sus", Val: "amogus"}}}

	out := bytes.Buffer{}

	isDone := executor.RunScript(context.Background(), nil, &out, nil, "echo -n $sus")
	<-isDone

	if out.String() != "amogus" {
		t.Fatalf("output must be \"amogus\", got \"%s\"", out.String())
	}
}

func TestRunScriptInput(t *testing.T) {
	executor := Executor{}

	in := "amogus"
	out := bytes.Buffer{}

	isDone := executor.RunScript(context.Background(), strings.NewReader(in), &out, nil, "cat -")
	<-isDone

	if out.String() != "amogus" {
		t.Fatalf("output must be \"amogus\", got \"%s\"", out.String())
	}
}
