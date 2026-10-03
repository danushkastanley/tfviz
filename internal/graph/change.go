package graph

import (
	"strings"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/projection"
	"github.com/danushkastanley/tfviz/internal/provider/aws"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// classify maps producer actions to a report change (plan §8). Moves and
// imports are annotations, never destroy-and-create.
func classify(r input.Resource, adapter aws.Adapter, plan bool) model.Change {
	if !plan {
		return model.Change{Action: model.ActionNoOp, ProducerActions: []string{}}
	}
	change := model.Change{
		Action:          actionOf(r.Actions),
		ProducerActions: clipActions(r.Actions),
		Reason:          clipReason(r.ActionReason),
	}
	switch strings.Join(r.Actions, ",") {
	case "delete,create":
		change.ReplaceOrder = model.ReplaceDeleteFirst
	case "create,delete":
		change.ReplaceOrder = model.ReplaceCreateFirst
	}
	if change.Action == model.ActionReplace {
		change.ReplaceFields, change.ReplaceFieldsOmitted = replaceFields(r.ReplacePaths, adapter)
	}
	if r.Importing {
		change.Annotations = append(change.Annotations, model.AnnotationImported)
	}
	if r.PreviousAddress != "" && r.PreviousAddress != r.Address {
		change.Annotations = append(change.Annotations, model.AnnotationMoved)
		change.PreviousAddress = r.PreviousAddress
	}
	return change
}

func actionOf(actions []string) model.Action {
	switch strings.Join(actions, ",") {
	case "no-op":
		return model.ActionNoOp
	case "create":
		return model.ActionCreate
	case "update":
		return model.ActionUpdate
	case "delete":
		return model.ActionDelete
	case "delete,create", "create,delete":
		return model.ActionReplace
	case "read":
		return model.ActionRead
	case "forget":
		return model.ActionForget
	default:
		return model.ActionUnsupported
	}
}

// replaceFields names only approved, non-secret fields that force
// replacement; any others are counted so the report stays honest.
func replaceFields(paths []string, adapter aws.Adapter) ([]string, int) {
	approved := map[string]bool{}
	for _, f := range adapter.Fields {
		if !f.Withheld {
			approved[f.Key] = true
		}
	}
	var named []string
	omitted := 0
	for _, path := range paths {
		if approved[path] && !projection.Denied(path) {
			named = append(named, path)
		} else {
			omitted++
		}
	}
	return named, omitted
}

// clipActions keeps producer action names short and printable; unknown
// actions are shown verbatim as evidence, so they must be bounded.
func clipActions(actions []string) []string {
	out := make([]string, 0, min(len(actions), 4))
	for i, a := range actions {
		if i == 4 {
			break
		}
		out = append(out, clipToken(a, 32))
	}
	return out
}

func clipReason(reason string) string { return clipToken(reason, 64) }

func clipToken(s string, limit int) string {
	var b strings.Builder
	for _, r := range s {
		if b.Len() >= limit {
			break
		}
		if r == '_' || r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
