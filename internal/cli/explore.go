package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/html"
	"github.com/danushkastanley/tfviz/internal/report/model"
	"github.com/danushkastanley/tfviz/internal/report/serve"
)

type exploreOptions struct {
	report reportOptions
	port   int
	open   bool
}

func parseExploreFlags(args []string, env Env) (exploreOptions, error) {
	var opts exploreOptions
	fs := flag.NewFlagSet("tfviz explore", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	r := &opts.report
	fs.StringVar(&r.input, "state", "", "state: show -json output, a version-4 .tfstate file, or s3://bucket/key (required)")
	fs.StringVar(&r.title, "title", "", "report title")
	fs.StringVar(&r.view, "view", "architecture", "structure the explorer opens in: architecture or modules")
	fs.BoolVar(&r.safeShare, "safe-share", false, "replace names, IDs, ARNs, accounts, addresses and network details with stand-ins")
	fs.BoolVar(&r.offline, "offline", false, "refuse any input that would need network access")
	fs.StringVar(&r.s3.profile, "aws-profile", "", "AWS profile for s3:// state")
	fs.StringVar(&r.s3.region, "aws-region", "", "region of the state bucket")
	fs.StringVar(&r.s3.version, "s3-version", "", "read this version of the s3:// object")
	fs.StringVar(&r.s3.owner, "expected-bucket-owner", "", "fail unless the s3:// bucket belongs to this account ID")
	fs.IntVar(&opts.port, "port", 0, "loopback port to listen on (default: any free port)")
	fs.BoolVar(&opts.open, "open", false, "open the explorer in your default browser")
	fs.Usage = func() {
		fmt.Fprintf(env.Stderr, "Usage: tfviz explore --state <file|s3://bucket/key> [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return opts, err
		}
		return opts, &usageError{"invalid options"}
	}
	switch {
	case fs.NArg() > 0:
		return opts, &usageError{fmt.Sprintf("unexpected argument %q", clip(fs.Arg(0)))}
	case r.input == "" || r.input == "-":
		return opts, &usageError{"--state is required: a state file or s3://bucket/key"}
	case r.view != "architecture" && r.view != "modules":
		return opts, &usageError{"--view must be architecture or modules"}
	case opts.port < 0 || opts.port > 65535:
		return opts, &usageError{"--port must be between 0 and 65535"}
	}
	return opts, nil
}

// runExplore serves the state on a loopback address until interrupted. It
// reads the snapshot once and again only when the viewer presses Refresh.
func runExplore(args []string, env Env) int {
	opts, err := parseExploreFlags(args, env)
	if errors.Is(err, flag.ErrHelp) {
		return ExitOK
	}
	if err != nil {
		fmt.Fprintf(env.Stderr, "tfviz: %v\n", err)
		return exitCode(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	load := func(context.Context) (*model.Report, error) {
		return buildReport(input.KindState, opts.report, env)
	}
	srv, err := serve.New(ctx, load, html.BundledAssets())
	if err != nil {
		fmt.Fprintf(env.Stderr, "tfviz: %v\n", err)
		return exitCode(err)
	}
	ln, err := srv.Listen(opts.port)
	if err != nil {
		fmt.Fprintf(env.Stderr, "tfviz: %v\n", err)
		return ExitFailure
	}
	link := srv.URL(ln.Addr().String())
	fmt.Fprintf(env.Stderr, "Exploring on this machine only. Open:\n  %s\nPress Ctrl+C to stop. The link works only while tfviz is running.\n", link)
	if opts.open {
		if err := openBrowser(link); err != nil {
			fmt.Fprintln(env.Stderr, "tfviz: could not open a browser; open the link above yourself.")
		}
	}
	if err := srv.Serve(ctx, ln); err != nil {
		fmt.Fprintf(env.Stderr, "tfviz: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintln(env.Stderr, "Explorer stopped.")
	return ExitOK
}

func openBrowser(link string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", link).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", link).Start()
	default:
		return exec.Command("xdg-open", link).Start()
	}
}
