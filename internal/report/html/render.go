// Package html renders a sanitised report model into one self-contained,
// offline HTML file.
package html

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/danushkastanley/tfviz/internal/report/model"
)

//go:embed report.html.tmpl
var pageSource string

var page = template.Must(template.New("report").Parse(pageSource))

type pageData struct {
	Title   string
	CSP     string
	Style   template.CSS
	Script  template.JS
	Data    template.JS
	Notices string
	Summary summaryView
}

// Render writes report as a self-contained HTML document. The report model
// is the only data embedded; it is serialised with HTML-significant
// characters escaped, so labels cannot break out of the data element.
func Render(w io.Writer, report *model.Report, assets Assets) error {
	if err := assets.validate(); err != nil {
		return err
	}
	data, err := marshalData(report)
	if err != nil {
		return err
	}
	view := pageData{
		Title:   report.Title,
		CSP:     contentSecurityPolicy(assets),
		Style:   template.CSS(assets.Style), // #nosec G203 -- validated bundle built from this repository.
		Script:  template.JS(assets.Script), // #nosec G203 -- validated bundle built from this repository.
		Data:    template.JS(data),          // #nosec G203 -- json.Marshal output with <, >, & and U+2028/9 escaped.
		Notices: string(assets.Notices),
		Summary: summarise(report),
	}
	var buf bytes.Buffer
	if err := page.Execute(&buf, view); err != nil {
		return fmt.Errorf("render report: %w", err)
	}
	_, err = buf.WriteTo(w)
	return err
}

// marshalData serialises the report for a <script type="application/json">
// element. encoding/json escapes <, > and & as < etc. and the JavaScript
// line terminators U+2028/U+2029, so no value can close the element or open
// a comment.
func marshalData(report *model.Report) ([]byte, error) {
	data, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode report: %w", err)
	}
	if bytes.Contains(data, []byte("<")) {
		return nil, fmt.Errorf("encode report: unescaped markup in report data")
	}
	return data, nil
}

// contentSecurityPolicy allows only the exact inlined script and stylesheet
// and denies every network fetch, frame, form, plugin and base override.
func contentSecurityPolicy(assets Assets) string {
	directives := []string{
		"default-src 'none'",
		"script-src " + hashSource(assets.Script),
		"style-src " + hashSource(assets.Style),
		"img-src 'none'",
		"font-src 'none'",
		"connect-src 'none'",
		"object-src 'none'",
		"base-uri 'none'",
		"form-action 'none'",
	}
	return strings.Join(directives, "; ")
}

func hashSource(content []byte) string {
	sum := sha256.Sum256(content)
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}
