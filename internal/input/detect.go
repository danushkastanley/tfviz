package input

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Read parses a plan or state JSON export. want is the kind the caller
// expects; a document of the other kind is rejected with guidance.
func Read(data []byte, want SnapshotKind) (*Snapshot, error) {
	if err := checkDepth(data); err != nil {
		if isStreamingUI(data) {
			return nil, streamingUIError()
		}
		return nil, err
	}
	var probe struct {
		FormatVersion   *string         `json:"format_version"`
		ResourceChanges json.RawMessage `json:"resource_changes"`
		PlannedValues   json.RawMessage `json:"planned_values"`
		Values          json.RawMessage `json:"values"`
		Version         json.RawMessage `json:"version"`
		Lineage         *string         `json:"lineage"`
		Level           *string         `json:"@level"`
		EncryptedData   json.RawMessage `json:"encrypted_data"`
		Meta            json.RawMessage `json:"meta"`
	}
	if err := decode(data, &probe); err != nil {
		if isStreamingUI(data) {
			return nil, streamingUIError()
		}
		return nil, err
	}
	switch {
	case probe.Level != nil:
		return nil, streamingUIError()
	case probe.EncryptedData != nil:
		return nil, newError(CodeEncrypted, "The input is an encrypted OpenTofu state or plan. Export it with `tofu show -json` using your encryption configuration, then pass that output to tfviz.")
	case probe.Lineage != nil && probe.Version != nil:
		return nil, newError(CodeRawState, "The input is a raw state file. Export it with `terraform show -json terraform.tfstate` (or `tofu show -json`) and pass that output to tfviz.")
	case probe.FormatVersion == nil:
		return nil, newError(CodeUnsupported, "The input is not a Terraform or OpenTofu JSON export. Create one with `terraform show -json` or `tofu show -json`.")
	}

	isPlan := probe.ResourceChanges != nil || probe.PlannedValues != nil
	switch {
	case isPlan && want == KindState:
		return nil, newError(CodeWrongKind, "The input is a plan. Use `tfviz plan` for plans.")
	case !isPlan && want == KindPlan:
		return nil, newError(CodeWrongKind, "The input is a state export, not a plan. Use `tfviz state` for state.")
	case isPlan:
		return readPlan(data)
	default:
		return readState(data)
	}
}

func streamingUIError() *Error {
	return newError(CodeStreamingUI, "The input looks like Terraform's streaming `-json` log output, not an exported plan. Save the plan with `-out`, then run `terraform show -json <planfile>`.")
}

// isStreamingUI recognises machine-readable UI logs: one JSON object per line
// with "@level" and "@message" keys.
func isStreamingUI(data []byte) bool {
	line, _, _ := bytes.Cut(bytes.TrimSpace(data), []byte("\n"))
	return bytes.Contains(line, []byte(`"@level"`)) && bytes.Contains(line, []byte(`"@message"`))
}

// checkFormat accepts major version 1. Newer minor versions are read with a
// notice because the format promises additive, compatible changes.
func checkFormat(version string, knownMinor int) ([]Notice, error) {
	major, minor, ok := strings.Cut(version, ".")
	if !ok || major != "1" {
		return nil, newError(CodeUnsupported, "This export uses JSON format version "+safeVersion(version)+", which tfviz does not support. Use Terraform 1.0 or later, or OpenTofu.")
	}
	if n := atoi(minor); n > knownMinor {
		return []Notice{{Code: NoticeUnknownFormatMinor, Message: "This export uses a newer JSON format (" + safeVersion(version) + ") than tfviz was tested with. Newer details may be missing."}}, nil
	}
	return nil, nil
}

// safeVersion keeps only version-like characters so a hostile document cannot
// smuggle text into messages.
func safeVersion(v string) string {
	var b strings.Builder
	for _, r := range v {
		if (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
		}
		if b.Len() >= 16 {
			break
		}
	}
	if b.Len() == 0 {
		return "(unrecognised)"
	}
	return b.String()
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
		if n > 1_000_000 {
			return n
		}
	}
	return n
}
