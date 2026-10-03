package input

import (
	"encoding/json"
	"strings"
	"testing"
)

func decodeAny(t *testing.T, s string) any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestBuildValueMarkers(t *testing.T) {
	raw := decodeAny(t, `{"name":"db","password":"hunter2","route":[{"cidr":"0.0.0.0/0"}],"tags":{"Env":"prod"}}`)
	sensitive := decodeAny(t, `{"password":true,"tags":true}`)
	unknown := decodeAny(t, `{"id":true,"route":[{"nat_gateway_id":true}],"endpoint":{"address":true}}`)
	v := buildValue(raw, sensitive, unknown, nil, nil)

	if s, _ := v.Field("name").String(); s != "db" {
		t.Fatalf("name = %q", s)
	}
	if v.Field("password").Kind() != KindSensitive || v.Field("password").Comparison() != ComparisonUnavailable {
		t.Fatal("password must be sensitive with no comparison when there is no other side")
	}
	if v.Field("tags").Kind() != KindSensitive || v.Field("tags").Field("Env").Kind() != KindSensitive {
		t.Fatal("a sensitive object hides every field beneath it")
	}
	if v.Field("id").Kind() != KindUnknown {
		t.Fatal("an attribute present only in unknown markers is unknown, not absent")
	}
	if v.Path("route", "nat_gateway_id").Kind() != KindUnknown {
		t.Fatal("unknown nested block attribute")
	}
	if c, _ := v.Path("route", "cidr").String(); c != "0.0.0.0/0" {
		t.Fatal("known sibling inside a partly unknown block is kept")
	}
	if v.Path("endpoint", "address").Kind() != KindUnknown {
		t.Fatal("an omitted, partly unknown structure is rebuilt from its markers")
	}
	if v.Field("missing").Kind() != KindNull {
		t.Fatal("unrecorded attributes are absent")
	}
}

func TestSensitiveComparison(t *testing.T) {
	before := decodeAny(t, `{"password":"old"}`)
	same := decodeAny(t, `{"password":"old"}`)
	changed := decodeAny(t, `{"password":"new"}`)
	marks := decodeAny(t, `{"password":true}`)
	if c := buildValue(same, marks, nil, &before, nil).Field("password").Comparison(); c != ComparisonUnchanged {
		t.Fatalf("same value: %v", c)
	}
	if c := buildValue(changed, marks, nil, &before, nil).Field("password").Comparison(); c != ComparisonChanged {
		t.Fatalf("changed value: %v", c)
	}
	unknownAfter := decodeAny(t, `{"password":true}`)
	if c := buildValue(before, marks, nil, &changed, unknownAfter).Field("password").Comparison(); c != ComparisonUnavailable {
		t.Fatalf("comparison against an unknown value must be unavailable: %v", c)
	}
}

func TestMergeMarks(t *testing.T) {
	merged := mergeMarks(decodeAny(t, `{"a":true,"b":{"c":false}}`), decodeAny(t, `{"b":{"c":true},"d":[false,true]}`))
	v := buildValue(decodeAny(t, `{"a":"x","b":{"c":"y"},"d":["p","q"]}`), merged, nil, nil, nil)
	if v.Field("a").Kind() != KindSensitive || v.Path("b", "c").Kind() != KindSensitive {
		t.Fatal("a mark on either side applies")
	}
	items := v.Field("d").Items()
	if items[0].Kind() != KindString || items[1].Kind() != KindSensitive {
		t.Fatal("list element marks are positional")
	}
}
