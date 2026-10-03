// Package projection turns reader values into approved report metadata.
//
// Every exported value passes through an explicit Field. Sensitivity always
// wins over the allowlist, unknown values stay unknown, and a deny guard
// withholds secret-shaped fields even if an adapter lists one by mistake.
package projection

import (
	"strings"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// Formatter derives a display value from a known, non-sensitive value. It is
// called only after the projection has handled sensitive, unknown and absent
// values, so it can never see what those markers hide.
type Formatter func(v input.Value) model.FieldValue

// Field approves one piece of metadata for export.
type Field struct {
	// Key is the stable report key, e.g. "broker_node_group_info.instance_type".
	Key string
	// Label is the reader-facing name.
	Label string
	// Path locates the value; it defaults to Key split on dots.
	Path []string
	// Format derives a display from structured values. Plain scalars and
	// string lists need none.
	Format Formatter
	// Inputs, when set, makes Format a whole-resource formatter that may read
	// only these sibling paths. Markers are checked on exactly these inputs.
	Inputs []string
	// ChangeOnly fields are secrets whose value is never exported but whose
	// change matters to a reviewer, such as a rotated password. They are
	// always reported as sensitive, with a change status only.
	ChangeOnly bool
	// Withheld fields are always reported as omitted: the attribute exists but
	// is deliberately not exported (tags, descriptions, usernames, bodies).
	Withheld bool
}

func (f Field) path() []string {
	if len(f.Path) > 0 {
		return f.Path
	}
	return strings.Split(f.Key, ".")
}

// deniedSegments are attribute names that hold secrets or opaque documents.
// They are withheld even when an adapter lists them, as a guard against
// allowlist mistakes and producers that fail to mark sensitive values.
var deniedSegments = map[string]bool{
	"password": true, "master_password": true, "passwd": true, "secret": true,
	"secret_string": true, "secret_binary": true, "token": true, "auth_token": true,
	"private_key": true, "user_data": true, "user_data_base64": true,
	"certificate_body": true, "certificate_chain": true, "environment": true,
	"variables": true, "container_definitions": true, "policy": true,
	"insecure_value": true, "value": true, "plaintext": true, "seed": true,
	"credentials": true, "connection_string": true,
}

func (f Field) denied() bool {
	if Denied(f.Key) || Denied(strings.Join(f.Path, ".")) {
		return true
	}
	for _, in := range f.Inputs {
		if Denied(in) {
			return true
		}
	}
	return false
}

// Denied reports whether a field key names secret-bearing content.
func Denied(key string) bool {
	for _, segment := range strings.Split(key, ".") {
		if deniedSegments[segment] {
			return true
		}
	}
	return false
}
