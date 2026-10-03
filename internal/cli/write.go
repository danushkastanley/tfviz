package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// checkOutput refuses to replace an existing file without --force, before
// any input is read.
func checkOutput(path string, force bool) error {
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("cannot check the output path: %w", errors.Unwrap(err))
	case info.IsDir():
		return &usageError{"--output is a directory; give a file path"}
	case !force:
		return &usageError{fmt.Sprintf("%s already exists; use --force to replace it", path)}
	case !info.Mode().IsRegular():
		return &usageError{"--output exists and is not a regular file; refusing to replace it"}
	}
	return nil
}

// writeAtomically writes to a private temporary file beside the destination
// and renames it into place, so readers never see a partial report. The
// report is readable only by its owner (0600).
func writeAtomically(path string, data []byte, force bool) error {
	if err := checkOutput(path, force); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tfviz-*.html.tmp")
	if err != nil {
		return fmt.Errorf("cannot write the report: %w", errors.Unwrap(err))
	}
	defer os.Remove(tmp.Name()) // removes only the temporary file this run created
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("cannot write the report: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("cannot write the report: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("cannot write the report: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("cannot write the report: %w", errors.Unwrap(err))
	}
	return nil
}
