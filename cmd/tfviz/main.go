// Command tfviz turns Terraform and OpenTofu plans and state into an
// offline HTML report.
package main

import (
	"os"
	"runtime/debug"
	"time"

	"github.com/danushkastanley/tfviz/internal/cli"
)

// version is set at release build time with -ldflags "-X main.version=…".
var version = ""

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Env{
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Now:     time.Now,
		Version: resolveVersion(),
	}))
}

func resolveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
