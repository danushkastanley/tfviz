package input

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

const (
	// MaxInputBytes bounds every input document. Large estates fit well within it.
	MaxInputBytes = 512 << 20
	// MaxDepth bounds JSON nesting; real producer output stays far below it.
	MaxDepth = 128
	// MaxResources bounds resource entries in one document.
	MaxResources = 50_000
)

// ReadBounded reads at most MaxInputBytes from r.
func ReadBounded(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxInputBytes+1))
	if err != nil {
		return nil, newError(CodeRead, "The input could not be read.")
	}
	if len(data) > MaxInputBytes {
		return nil, newError(CodeTooLarge, "The input is larger than 512 MiB.")
	}
	return data, nil
}

// checkDepth walks the token stream once to reject excessive nesting before
// a full decode allocates the document.
func checkDepth(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	depth := 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return newError(CodeMalformed, "The input is not valid JSON.")
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{', '[':
				depth++
				if depth > MaxDepth {
					return newError(CodeTooDeep, "The input is nested too deeply to be a Terraform or OpenTofu export.")
				}
			case '}', ']':
				depth--
			}
		}
	}
}

func decode(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(v); err != nil {
		return newError(CodeMalformed, "The input is not a valid Terraform or OpenTofu JSON export.")
	}
	if dec.More() {
		return newError(CodeMalformed, "The input contains more than one JSON document. Export a single plan or state with show -json.")
	}
	return nil
}

func equalJSON(a, b any) bool {
	ea, errA := json.Marshal(a)
	eb, errB := json.Marshal(b)
	return errA == nil && errB == nil && bytes.Equal(ea, eb)
}
