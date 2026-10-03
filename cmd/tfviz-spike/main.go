// Command tfviz-spike renders an existing report model JSON file into the
// offline HTML report. It exists only for the M0 export spike and is removed
// when the real CLI lands in M1.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/danushkastanley/tfviz/internal/report/html"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

const maxReportBytes = 64 << 20

func main() {
	in := flag.String("in", "", "report model JSON (schema v1)")
	out := flag.String("out", "", "HTML file to write")
	force := flag.Bool("force", false, "replace an existing output file")
	flag.Parse()
	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: tfviz-spike -in report.json -out report.html [-force]")
		os.Exit(2)
	}
	if err := run(*in, *out, *force); err != nil {
		fmt.Fprintln(os.Stderr, "tfviz-spike:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Wrote %s\n", *out)
}

func run(in, out string, force bool) error {
	raw, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	if len(raw) > maxReportBytes {
		return errors.New("report model is larger than 64 MiB")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var report model.Report
	if err := dec.Decode(&report); err != nil {
		return fmt.Errorf("read report model: %w", err)
	}
	var page bytes.Buffer
	if err := html.Render(&page, &report, html.BundledAssets()); err != nil {
		return err
	}
	return writeAtomically(out, page.Bytes(), force)
}

// writeAtomically writes to a private temporary file beside the destination
// and renames it into place, so readers never see a partial report.
func writeAtomically(path string, data []byte, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("%s already exists; use -force to replace it", path)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tfviz-*.html.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
