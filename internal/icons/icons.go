// Package icons embeds service icons from a pack the user downloaded and
// pointed tfviz at with --icons. tfviz never ships provider icons: the
// providers allow their use in architecture diagrams, which is what the user
// makes with them, but not redistribution inside software.
package icons

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/danushkastanley/tfviz/internal/report/model"
)

const (
	maxFileBytes  = 64 << 10 // the largest official AWS icon is about 14 KiB
	maxTotalBytes = 2 << 20
	maxFiles      = 50000 // the AWS pack holds about 8,300 files
)

// sizes are the pack's size suffixes in order of preference. Every size is
// a vector image, so the choice only affects line weight.
var sizes = []string{"48", "64", "32", "16"}

// Error explains why an icon folder cannot be used. Its message names only
// the folder or a file in it, never file contents.
type Error struct{ msg string }

func (e *Error) Error() string { return e.msg }

// Load reads the icons for types from dir. stems lists, per resource type,
// file names without the size suffix or extension, most specific first.
// It fails if dir holds none of the listed icons, because that almost
// always means the wrong folder. It returns nil if dir is usable but none
// of types has an icon.
func Load(dir string, stems map[string][]string, types []string) (*model.Icons, error) {
	files, err := index(dir)
	if err != nil {
		return nil, err
	}
	if !anyKnown(files, stems) {
		return nil, &Error{fmt.Sprintf("--icons: no recognised icons in %s; use the unzipped AWS Architecture Icons folder", dir)}
	}
	icons := &model.Icons{Images: map[string]string{}, ByType: map[string]string{}}
	ids := map[string]string{} // path → icon id, so shared icons are embedded once
	total := 0
	sorted := append([]string(nil), types...)
	sort.Strings(sorted)
	for _, typ := range sorted {
		path := find(files, stems[typ])
		if path == "" {
			continue
		}
		if id, ok := ids[path]; ok {
			icons.ByType[typ] = id
			continue
		}
		uri, err := dataURI(path)
		if err != nil {
			return nil, err
		}
		if total += len(uri); total > maxTotalBytes {
			return nil, &Error{"--icons: the matching icons are too large to embed in one report"}
		}
		id := fmt.Sprintf("ic%d", len(ids)+1)
		ids[path] = id
		icons.Images[id] = uri
		icons.ByType[typ] = id
	}
	if len(icons.ByType) == 0 {
		return nil, nil
	}
	return icons, nil
}

// index maps each SVG file's name, without extension, to its path. It skips
// hidden entries, macOS archive metadata and anything that is not a regular
// file, so symbolic links cannot pull in files from elsewhere.
func index(dir string) (map[string]string, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, &Error{fmt.Sprintf("--icons: %s is not a folder; unzip the icon pack and pass its folder", dir)}
	}
	files := map[string]string{}
	count := 0
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if path != dir && (strings.HasPrefix(name, ".") || name == "__MACOSX") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if count++; count > maxFiles {
			return errTooMany
		}
		if d.Type().IsRegular() && strings.HasSuffix(name, ".svg") {
			stem := strings.TrimSuffix(name, ".svg")
			if _, seen := files[stem]; !seen {
				files[stem] = path
			}
		}
		return nil
	})
	if errors.Is(err, errTooMany) {
		return nil, &Error{fmt.Sprintf("--icons: %s holds more than %d files; point it at the icon pack folder", dir, maxFiles)}
	}
	if err != nil {
		return nil, &Error{fmt.Sprintf("--icons: could not read %s", dir)}
	}
	return files, nil
}

var errTooMany = errors.New("too many files")

func anyKnown(files map[string]string, stems map[string][]string) bool {
	for _, list := range stems {
		if find(files, list) != "" {
			return true
		}
	}
	return false
}

func find(files map[string]string, stems []string) string {
	for _, stem := range stems {
		for _, size := range sizes {
			if path, ok := files[stem+"_"+size]; ok {
				return path
			}
		}
	}
	return ""
}

// dataURI reads one icon and checks that it is an SVG document. Reports
// show icons only through <img>, where browsers run no scripts and fetch
// nothing, so the check is about catching the wrong file, not sanitising.
func dataURI(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", &Error{fmt.Sprintf("--icons: could not read %s", path)}
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	switch {
	case err != nil:
		return "", &Error{fmt.Sprintf("--icons: could not read %s", path)}
	case len(data) > maxFileBytes:
		return "", &Error{fmt.Sprintf("--icons: %s is larger than %d KiB", path, maxFileBytes>>10)}
	case !isSVG(data):
		return "", &Error{fmt.Sprintf("--icons: %s is not an SVG image", path)}
	}
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(data), nil
}

func isSVG(data []byte) bool {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			return false
		}
		if start, ok := tok.(xml.StartElement); ok {
			return start.Name.Local == "svg"
		}
	}
}
