package icons

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const svg = `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48"><rect width="48" height="48"/></svg>`

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func decode(t *testing.T, uri string) string {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, "data:image/svg+xml;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestLoadPrefersSpecificIconsAndSize48(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "Arch/64/Arch_Svc_64.svg", strings.Replace(svg, "48 48", "64 64", 1))
	write(t, dir, "Arch/48/Arch_Svc_48.svg", svg)
	write(t, dir, "Res/Res_Svc_Thing_32.svg", strings.Replace(svg, "48 48", "32 32", 1))
	stems := map[string][]string{
		"t_thing":   {"Res_Svc_Thing", "Arch_Svc"},
		"t_service": {"Arch_Svc"},
		"t_other":   {"Arch_Svc"},
		"t_missing": {"Arch_Nothing"},
	}
	got, err := Load(dir, stems, []string{"t_service", "t_thing", "t_other", "t_missing", "t_unmapped"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(decode(t, got.Images[got.ByType["t_thing"]]), "32 32") {
		t.Error("t_thing should use its resource icon")
	}
	if !strings.Contains(decode(t, got.Images[got.ByType["t_service"]]), "48 48") {
		t.Error("t_service should prefer the 48 size")
	}
	if got.ByType["t_service"] != got.ByType["t_other"] || len(got.Images) != 2 {
		t.Errorf("shared icons should be embedded once: %v", got.ByType)
	}
	if _, ok := got.ByType["t_missing"]; ok {
		t.Error("types without a matching file should have no icon")
	}
}

func TestLoadIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	stems := map[string][]string{}
	var types []string
	for i := range 20 {
		name := fmt.Sprintf("Arch_S%d", i)
		write(t, dir, name+"_48.svg", svg)
		typ := fmt.Sprintf("t_%02d", i)
		stems[typ] = []string{name}
		types = append(types, typ)
	}
	first, _ := Load(dir, stems, types)
	reversed := append([]string(nil), types...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	second, _ := Load(dir, stems, reversed)
	if fmt.Sprint(first.ByType) != fmt.Sprint(second.ByType) || first.ByType["t_00"] != "ic1" {
		t.Errorf("ids depend on input order: %v vs %v", first.ByType, second.ByType)
	}
}

func TestLoadSkipsMetadataHiddenFilesAndLinks(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	write(t, outside, "Arch_Svc_48.svg", svg)
	write(t, dir, "__MACOSX/Arch_Svc_48.svg", svg)
	write(t, dir, ".cache/Arch_Svc_48.svg", svg)
	if err := os.Symlink(filepath.Join(outside, "Arch_Svc_48.svg"), filepath.Join(dir, "Arch_Svc_48.svg")); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir, map[string][]string{"t": {"Arch_Svc"}}, []string{"t"})
	if err == nil || !strings.Contains(err.Error(), "no recognised icons") {
		t.Fatalf("err = %v, want no recognised icons", err)
	}
}

func TestLoadReturnsNilWhenNoReportTypeHasAnIcon(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "Arch_Svc_48.svg", svg)
	got, err := Load(dir, map[string][]string{"t": {"Arch_Svc"}}, []string{"other"})
	if err != nil || got != nil {
		t.Fatalf("got %v, %v; want nil, nil", got, err)
	}
}

func TestLoadRejectsUnusableFolders(t *testing.T) {
	stems := map[string][]string{"t": {"Arch_Svc"}}
	file := filepath.Join(t.TempDir(), "pack.zip")
	write(t, filepath.Dir(file), "pack.zip", "PK")
	cases := map[string]struct {
		setup func(dir string)
		path  func(dir string) string
		want  string
	}{
		"missing":  {func(string) {}, func(dir string) string { return filepath.Join(dir, "nope") }, "is not a folder"},
		"zip file": {func(string) {}, func(string) string { return file }, "unzip the icon pack"},
		"empty":    {func(string) {}, func(dir string) string { return dir }, "no recognised icons"},
		"not svg": {func(dir string) { write(t, dir, "Arch_Svc_48.svg", "<html><body>x</body></html>") },
			func(dir string) string { return dir }, "is not an SVG image"},
		"too large": {func(dir string) { write(t, dir, "Arch_Svc_48.svg", svg+strings.Repeat(" ", maxFileBytes)) },
			func(dir string) string { return dir }, "larger than"},
	}
	for name, tc := range cases {
		dir := t.TempDir()
		tc.setup(dir)
		_, err := Load(tc.path(dir), stems, []string{"t"})
		var iconErr *Error
		if !errors.As(err, &iconErr) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tc.want)
		}
	}
}

func TestLoadCapsTheTotalSize(t *testing.T) {
	dir := t.TempDir()
	big := strings.Replace(svg, "</svg>", "<!--"+strings.Repeat("x", 60<<10)+"--></svg>", 1)
	stems := map[string][]string{}
	var types []string
	for i := range 40 {
		write(t, dir, fmt.Sprintf("Arch_S%d_48.svg", i), big)
		typ := fmt.Sprintf("t%d", i)
		stems[typ] = []string{fmt.Sprintf("Arch_S%d", i)}
		types = append(types, typ)
	}
	if _, err := Load(dir, stems, types); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("err = %v, want too large", err)
	}
}
