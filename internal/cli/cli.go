// Package cli implements the tfviz command line.
package cli

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/danushkastanley/tfviz/internal/input"
)

// Exit codes (plan §5). A plan that changes infrastructure is a success.
const (
	ExitOK          = 0
	ExitFailure     = 1 // processing or retrieval failed
	ExitUnsupported = 2 // invalid arguments or unsupported input
)

// Env is the process environment, injected for tests.
type Env struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Now     func() time.Time
	Version string
	// S3Endpoint overrides the S3 endpoint; tests only.
	S3Endpoint string
}

// Run executes tfviz with arguments (excluding the program name) and
// returns the process exit code.
func Run(args []string, env Env) int {
	if len(args) == 0 {
		fmt.Fprint(env.Stderr, usage)
		return ExitUnsupported
	}
	switch args[0] {
	case "plan":
		return runReport(input.KindPlan, args[1:], env)
	case "state":
		return runReport(input.KindState, args[1:], env)
	case "version", "--version":
		fmt.Fprintf(env.Stdout, "tfviz %s\n", env.Version)
		return ExitOK
	case "help", "-h", "--help":
		fmt.Fprint(env.Stdout, usage)
		return ExitOK
	default:
		fmt.Fprintf(env.Stderr, "tfviz: unknown command %q\n\n%s", clip(args[0]), usage)
		return ExitUnsupported
	}
}

const usage = `tfviz turns Terraform and OpenTofu plans and state into an offline HTML report.

Usage:
  terraform show -json tfplan | tfviz plan --input - --output infra-report.html
  terraform show -json        | tfviz state --input - --output state-report.html

Commands:
  plan      Generate a review report from an exported plan (show -json)
  state     Generate a report from exported state (show -json)
  version   Print the version

Run "tfviz <command> --help" for options.
`

// usageError is an invalid invocation (exit code 2).
type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

// exitCode classifies an error: unsupported input and invalid arguments are
// exit code 2; everything else, including S3 retrieval, is a processing failure.
func exitCode(err error) int {
	var inputErr *input.Error
	var use *usageError
	switch {
	case errors.As(err, &use):
		return ExitUnsupported
	case errors.As(err, &inputErr) && inputErr.Unsupported():
		return ExitUnsupported
	default:
		return ExitFailure
	}
}

// clip bounds user-supplied text echoed in messages.
func clip(s string) string {
	const limit = 64
	if len(s) > limit {
		return s[:limit] + "…"
	}
	return s
}
