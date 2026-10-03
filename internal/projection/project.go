package projection

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

const (
	maxText      = 2048
	maxListItems = 256
)

// Project exports the approved fields of one resource. In plan mode each
// field carries a change status; in state mode only the current value.
func Project(r input.Resource, fields []Field, plan bool) []model.MetadataField {
	out := make([]model.MetadataField, 0, len(fields))
	for _, f := range fields {
		out = append(out, projectField(r, f, plan))
	}
	return out
}

func projectField(r input.Resource, f Field, plan bool) model.MetadataField {
	field := model.MetadataField{Key: f.Key, Label: f.Label}
	if f.ChangeOnly {
		return changeOnly(r, f, plan)
	}
	if f.Withheld || f.denied() {
		v := model.Omitted()
		field.After = &v
		if plan {
			field.ChangeStatus = model.ChangeUnchanged
		}
		return field
	}
	if !plan {
		v := f.display(r.After)
		field.After = &v
		return field
	}

	before := f.locate(r.Before)
	after := f.locate(r.After)
	var b, a *model.FieldValue
	if r.HasBefore {
		v := f.display(r.Before)
		b = &v
	}
	if r.HasAfter {
		v := f.display(r.After)
		a = &v
	}
	field.ChangeStatus = changeStatus(b, a, before, after)
	if field.ChangeStatus == model.ChangeUnchanged {
		field.After = a
	} else {
		field.Before, field.After = b, a
	}
	return field
}

// changeOnly reports a secret as sensitive, never with a payload. Its change
// comes from the reader's comparison (for marked values) or from comparing
// known values here, which discloses only whether they differ.
func changeOnly(r input.Resource, f Field, plan bool) model.MetadataField {
	field := model.MetadataField{Key: f.Key, Label: f.Label}
	sensitive := model.Sensitive()
	if !plan {
		field.After = &sensitive
		return field
	}
	before, after := r.Before.Path(f.path()...), r.After.Path(f.path()...)
	switch {
	case !r.HasBefore && r.HasAfter:
		field.After, field.ChangeStatus = &sensitive, model.ChangeAdded
	case r.HasBefore && !r.HasAfter:
		field.Before, field.ChangeStatus = &sensitive, model.ChangeRemoved
	default:
		field.Before, field.After = &sensitive, &sensitive
		field.ChangeStatus = secretChange(before, after)
		if field.ChangeStatus == model.ChangeUnchanged {
			field.Before = nil
		}
	}
	return field
}

func secretChange(before, after input.Value) model.ChangeStatus {
	if before.Kind() == input.KindSensitive || after.Kind() == input.KindSensitive {
		return sensitiveChange(before, after)
	}
	if before.Kind() == input.KindUnknown || after.Kind() == input.KindUnknown {
		return model.ChangeUnknown
	}
	b, bok := before.String()
	a, aok := after.String()
	switch {
	case bok && aok && a == b, before.Kind() == input.KindNull && after.Kind() == input.KindNull:
		return model.ChangeUnchanged
	case bok || aok:
		return model.ChangeChanged
	default:
		return model.ChangeUnknown
	}
}

// locate returns the value the field reads: the attribute itself, or for a
// multi-input formatter the first input that is sensitive or unknown (so
// change detection sees the marker), otherwise the first input.
func (f Field) locate(root input.Value) input.Value {
	if len(f.Inputs) == 0 {
		return root.Path(f.path()...)
	}
	for _, in := range f.Inputs {
		if v := root.Path(strings.Split(in, ".")...); v.Kind() == input.KindSensitive || v.Kind() == input.KindUnknown {
			return v
		}
	}
	return root.Path(strings.Split(f.Inputs[0], ".")...)
}

func (f Field) display(root input.Value) model.FieldValue {
	if len(f.Inputs) == 0 {
		return display(root.Path(f.path()...), f.Format)
	}
	for _, in := range f.Inputs {
		v := root.Path(strings.Split(in, ".")...)
		switch {
		case v.Kind() == input.KindSensitive || v.Contains(input.KindSensitive):
			return model.Sensitive()
		case v.Kind() == input.KindUnknown || v.Contains(input.KindUnknown):
			return model.Unknown()
		}
	}
	if root.Kind() == input.KindNull {
		return model.Absent()
	}
	return f.Format(root)
}

// display maps a reader value to a report value. Sensitive and unknown
// markers win over any formatter, including when nested inside the value.
func display(v input.Value, format Formatter) model.FieldValue {
	switch {
	case v.Kind() == input.KindSensitive:
		return model.Sensitive()
	case v.Kind() == input.KindUnknown:
		return model.Unknown()
	case v.Kind() == input.KindNull:
		return model.Absent()
	case v.Contains(input.KindSensitive):
		return model.Sensitive()
	case v.Contains(input.KindUnknown):
		return model.Unknown()
	case format != nil:
		return format(v)
	}
	if s, ok := v.String(); ok {
		return model.KnownString(clip(s))
	}
	if n, ok := v.Number(); ok {
		return model.KnownNumber(n)
	}
	if b, ok := v.Bool(); ok {
		return model.KnownBool(b)
	}
	if items, ok := v.Strings(); ok {
		return List(items)
	}
	// Structured values need an explicit formatter; never dump them.
	return model.Omitted()
}

func changeStatus(before, after *model.FieldValue, rawBefore, rawAfter input.Value) model.ChangeStatus {
	switch {
	case before == nil && after == nil:
		return model.ChangeUnknown
	case before == nil:
		return model.ChangeAdded
	case after == nil:
		return model.ChangeRemoved
	case after.Status() == model.StatusUnknown:
		return model.ChangeUnknown
	case before.Status() == model.StatusSensitive || after.Status() == model.StatusSensitive:
		return sensitiveChange(rawBefore, rawAfter)
	case equal(*before, *after):
		return model.ChangeUnchanged
	default:
		return model.ChangeChanged
	}
}

// sensitiveChange uses the reader's verdict, established before the raw
// values were discarded. Without one the comparison is unavailable.
func sensitiveChange(before, after input.Value) model.ChangeStatus {
	for _, v := range []input.Value{after, before} {
		if v.Kind() != input.KindSensitive {
			continue
		}
		switch v.Comparison() {
		case input.ComparisonChanged:
			return model.ChangeChanged
		case input.ComparisonUnchanged:
			return model.ChangeUnchanged
		}
	}
	return model.ChangeUnknown
}

func equal(a, b model.FieldValue) bool {
	ea, errA := json.Marshal(a)
	eb, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(ea) == string(eb)
}

// List builds a bounded list display value.
func List(items []string) model.FieldValue {
	if len(items) > maxListItems {
		items = append(items[:maxListItems-1:maxListItems-1], "…")
	}
	clipped := make([]string, len(items))
	for i, s := range items {
		clipped[i] = clip(s)
	}
	return model.KnownList(clipped)
}

func clip(s string) string {
	if len(s) <= maxText {
		return s
	}
	cut := maxText - len("…")
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}
