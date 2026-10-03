package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/build"
	"github.com/danushkastanley/tfviz/internal/report/html"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

type reportOptions struct {
	input  string
	output string
	title  string
	view   string
	force  bool
	strict bool
}

func parseReportFlags(kind input.SnapshotKind, args []string, env Env) (reportOptions, error) {
	var opts reportOptions
	fs := flag.NewFlagSet("tfviz "+string(kind), flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	noun := map[input.SnapshotKind]string{input.KindPlan: "plan", input.KindState: "state"}[kind]
	fs.StringVar(&opts.input, "input", "", "exported "+noun+" JSON file, or - for standard input (required)")
	fs.StringVar(&opts.output, "output", "", "HTML report to write (required)")
	fs.StringVar(&opts.title, "title", "", "report title")
	fs.StringVar(&opts.view, "view", "architecture", "structure the report opens in: architecture or modules")
	fs.BoolVar(&opts.force, "force", false, "replace the output file if it exists")
	fs.BoolVar(&opts.strict, "strict", false, "fail if any resource, action or input cannot be fully interpreted")
	fs.Usage = func() {
		fmt.Fprintf(env.Stderr, "Usage: tfviz %s --input <file|-> --output <report.html> [options]\n\nOptions:\n", kind)
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
	case opts.input == "":
		return opts, &usageError{"--input is required (use - for standard input)"}
	case opts.output == "":
		return opts, &usageError{"--output is required"}
	case opts.view != "architecture" && opts.view != "modules":
		return opts, &usageError{"--view must be architecture or modules"}
	case opts.output == "-":
		return opts, &usageError{"--output must be a file path; reports are not written to standard output"}
	}
	return opts, nil
}

func runReport(kind input.SnapshotKind, args []string, env Env) int {
	opts, err := parseReportFlags(kind, args, env)
	if errors.Is(err, flag.ErrHelp) {
		return ExitOK
	}
	if err != nil {
		fmt.Fprintf(env.Stderr, "tfviz: %v\n", err)
		return exitCode(err)
	}
	summary, err := generate(kind, opts, env)
	if err != nil {
		fmt.Fprintf(env.Stderr, "tfviz: %v\n", err)
		return exitCode(err)
	}
	fmt.Fprintln(env.Stderr, summary)
	return ExitOK
}

func generate(kind input.SnapshotKind, opts reportOptions, env Env) (string, error) {
	if err := checkOutput(opts.output, opts.force); err != nil {
		return "", err
	}
	data, source, err := readInput(opts.input, env.Stdin)
	if err != nil {
		return "", err
	}
	snap, err := input.Read(data, kind)
	data = nil // the raw, secret-bearing input is not needed beyond this point
	if err != nil {
		return "", err
	}
	report := build.Report(snap, build.Options{Title: opts.title, View: model.View(opts.view), Source: source, GeneratedAt: env.Now(), ToolVersion: env.Version})
	if opts.strict {
		if problems := strictProblems(report); len(problems) > 0 {
			return "", &usageError{"--strict: " + strings.Join(problems, "; ")}
		}
	}
	var page bytes.Buffer
	if err := html.Render(&page, report, html.BundledAssets()); err != nil {
		return "", err
	}
	if err := writeAtomically(opts.output, page.Bytes(), opts.force); err != nil {
		return "", err
	}
	return describe(opts.output, report), nil
}

func readInput(path string, stdin io.Reader) ([]byte, model.SourceKind, error) {
	if path == "-" {
		data, err := input.ReadBounded(stdin)
		return data, model.SourceStdin, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read the input file: %w", errors.Unwrap(err))
	}
	if !info.Mode().IsRegular() {
		return nil, "", &usageError{"the input must be a regular file or - for standard input"}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read the input file: %w", errors.Unwrap(err))
	}
	defer f.Close()
	data, err := input.ReadBounded(f)
	return data, model.SourceFile, err
}

// strictProblems lists interpretation gaps that --strict turns into failure.
func strictProblems(r *model.Report) []string {
	var problems []string
	if r.Coverage.ResourcesGeneric > 0 {
		problems = append(problems, fmt.Sprintf("%d unsupported resource(s)", r.Coverage.ResourcesGeneric))
	}
	if r.Summary.Unsupported > 0 {
		problems = append(problems, fmt.Sprintf("%d unrecognised action(s)", r.Summary.Unsupported))
	}
	for _, w := range r.Warnings {
		switch w.Code {
		case model.WarningIncompletePlan, model.WarningDeposedObjectSkipped, model.WarningUnknownFormatMinor:
			problems = append(problems, w.Message)
		}
	}
	return problems
}

// describe is the one-line result: counts only, never input content.
func describe(path string, r *model.Report) string {
	s := r.Summary
	line := fmt.Sprintf("Wrote %s: %d resources (%d fully supported", path, r.Coverage.ResourcesTotal, r.Coverage.ResourcesSupported)
	if r.Coverage.ResourcesGeneric > 0 {
		line += fmt.Sprintf(", %d with limited detail", r.Coverage.ResourcesGeneric)
	}
	line += ")"
	if r.Mode == model.ModePlan {
		line += fmt.Sprintf(". Changes: %d to add, %d to change, %d to replace, %d to destroy", s.Create, s.Update, s.Replace, s.Delete)
		if s.Read > 0 {
			line += fmt.Sprintf(", %d read during apply", s.Read)
		}
		if s.Forget > 0 {
			line += fmt.Sprintf(", %d removed from state", s.Forget)
		}
	}
	if n := len(r.Unresolved); n > 0 {
		line += fmt.Sprintf(". %d unresolved reference(s)", n)
	}
	return line + "."
}
