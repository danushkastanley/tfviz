package html

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
)

// The web bundle is built by `make web`, which copies web/dist here. It is not
// committed; building the Go binary without it fails at compile time.
var (
	//go:embed dist/tfviz.js
	bundledScript []byte
	//go:embed dist/tfviz.css
	bundledStyle []byte
	//go:embed dist/THIRD_PARTY_NOTICES.txt
	bundledNotices []byte
)

// Assets are the interface files inlined into every report.
type Assets struct {
	Script  []byte
	Style   []byte
	Notices []byte
}

// BundledAssets returns the interface compiled into this binary.
func BundledAssets() Assets {
	return Assets{Script: bundledScript, Style: bundledStyle, Notices: bundledNotices}
}

// validate refuses assets that could terminate their enclosing element or
// comment early. Inlined code is never escaped (its CSP hash must match the
// exact bytes), so the only safe option is to reject such a bundle outright.
func (a Assets) validate() error {
	if len(a.Script) == 0 || len(a.Style) == 0 {
		return errors.New("report interface assets are missing")
	}
	checks := []struct {
		name    string
		content []byte
		closer  string
	}{
		{"script", a.Script, "</script"},
		{"script", a.Script, "<!--"},
		{"stylesheet", a.Style, "</style"},
	}
	for _, c := range checks {
		if bytes.Contains(bytes.ToLower(c.content), []byte(c.closer)) {
			return fmt.Errorf("report interface %s contains %q and cannot be inlined safely", c.name, c.closer)
		}
	}
	return nil
}
