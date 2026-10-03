package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

type ValueStatus string

const (
	StatusKnown     ValueStatus = "known"
	StatusSensitive ValueStatus = "sensitive"
	StatusUnknown   ValueStatus = "unknown"
	StatusAbsent    ValueStatus = "absent"
	StatusOmitted   ValueStatus = "omitted"
)

// FieldValue carries a display payload only when its status is known. The
// type makes it impossible to attach a value to a sensitive field: there is
// no constructor for that, and marshalling drops any payload not marked known.
type FieldValue struct {
	status ValueStatus
	value  any // string, float64, bool or []string; nil unless known
}

func KnownString(s string) FieldValue  { return FieldValue{status: StatusKnown, value: s} }
func KnownNumber(n float64) FieldValue { return FieldValue{status: StatusKnown, value: n} }
func KnownBool(b bool) FieldValue      { return FieldValue{status: StatusKnown, value: b} }
func KnownList(items []string) FieldValue {
	return FieldValue{status: StatusKnown, value: append([]string{}, items...)}
}
func Sensitive() FieldValue { return FieldValue{status: StatusSensitive} }
func Unknown() FieldValue   { return FieldValue{status: StatusUnknown} }
func Absent() FieldValue    { return FieldValue{status: StatusAbsent} }
func Omitted() FieldValue   { return FieldValue{status: StatusOmitted} }

func (v FieldValue) Status() ValueStatus { return v.status }

// Value returns the display payload, or nil when the status is not known.
func (v FieldValue) Value() any { return v.value }

type fieldValueJSON struct {
	Status ValueStatus `json:"status"`
	Value  any         `json:"value,omitempty"`
}

func (v FieldValue) MarshalJSON() ([]byte, error) {
	if v.status == "" {
		return nil, errors.New("model: FieldValue has no status")
	}
	if v.status != StatusKnown {
		return json.Marshal(fieldValueJSON{Status: v.status})
	}
	// A known false or zero must still be emitted, so bypass omitempty.
	return json.Marshal(struct {
		Status ValueStatus `json:"status"`
		Value  any         `json:"value"`
	}{v.status, v.value})
}

func (v *FieldValue) UnmarshalJSON(data []byte) error {
	var raw struct {
		Status ValueStatus     `json:"status"`
		Value  json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch raw.Status {
	case StatusSensitive, StatusUnknown, StatusAbsent, StatusOmitted:
		if len(raw.Value) > 0 {
			return fmt.Errorf("model: %s value must not carry a payload", raw.Status)
		}
		*v = FieldValue{status: raw.Status}
		return nil
	case StatusKnown:
		return v.unmarshalKnown(raw.Value)
	default:
		return fmt.Errorf("model: unrecognised value status %q", raw.Status)
	}
}

func (v *FieldValue) unmarshalKnown(data json.RawMessage) error {
	// json.Unmarshal treats null as a successful no-op, so reject it first.
	if trimmed := string(bytes.TrimSpace(data)); trimmed == "" || trimmed == "null" {
		return errors.New("model: known value must carry a payload")
	}
	var s string
	if json.Unmarshal(data, &s) == nil {
		*v = KnownString(s)
		return nil
	}
	var n float64
	if json.Unmarshal(data, &n) == nil {
		*v = KnownNumber(n)
		return nil
	}
	var b bool
	if json.Unmarshal(data, &b) == nil {
		*v = KnownBool(b)
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil || list == nil {
		return errors.New("model: known value must be a string, number, boolean or list of strings")
	}
	*v = KnownList(list)
	return nil
}
