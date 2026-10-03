package input

import (
	"encoding/json"
	"testing"
)

func TestReadsRawState(t *testing.T) {
	for _, tt := range []struct {
		path     string
		producer Producer
	}{
		{"terraform-1.16/prior.tfstate", ProducerTerraform},
		{"opentofu-1.13/prior.tfstate", ProducerOpenTofu},
	} {
		snap := mustRead(t, tt.path, KindState)
		if snap.Producer != tt.producer || len(snap.Resources) != 35 || snap.FormatVersion != "raw-v4" {
			t.Fatalf("%s: producer %s, %d resources, format %s", tt.path, snap.Producer, len(snap.Resources), snap.FormatVersion)
		}
		db := resource(t, snap, "aws_db_instance.orders")
		if db.After.Field("password").Kind() != KindSensitive {
			t.Fatalf("%s: sensitive_attributes must mark the password", tt.path)
		}
		subnet := resource(t, snap, `aws_subnet.private["a"]`)
		if subnet.Provider != "hashicorp/aws" || subnet.Module != "" {
			t.Fatalf("%s: subnet %+v", tt.path, subnet)
		}
		if msk := resource(t, snap, "module.streaming.aws_msk_cluster.events"); msk.Module != "module.streaming" {
			t.Fatalf("%s: module = %q", tt.path, msk.Module)
		}
		if resource(t, snap, `aws_vpc_security_group_ingress_rule.msk_plaintext[0]`).Address == "" {
			t.Fatal("numeric instance keys are kept")
		}
	}
}

func TestSensitivityMarks(t *testing.T) {
	paths := []json.RawMessage{
		json.RawMessage(`[{"type":"get_attr","value":"password"}]`),
		json.RawMessage(`[{"type":"get_attr","value":"rules"},{"type":"index","value":{"value":1,"type":"number"}},{"type":"get_attr","value":"token"}]`),
		json.RawMessage(`[{"type":"get_attr","value":"tags"},{"type":"index","value":{"value":"Secret","type":"string"}}]`),
	}
	raw := decodeAny(t, `{"password":"p","rules":[{"token":"a"},{"token":"b"}],"tags":{"Secret":"s","Name":"n"}}`)
	v := buildValue(raw, sensitivityMarks(paths), nil, nil, nil)
	rules := v.Field("rules").Items()
	switch {
	case v.Field("password").Kind() != KindSensitive:
		t.Error("attribute path")
	case rules[0].Field("token").Kind() != KindString || rules[1].Field("token").Kind() != KindSensitive:
		t.Error("numeric index path")
	case v.Path("tags", "Secret").Kind() != KindSensitive || v.Path("tags", "Name").Kind() != KindString:
		t.Error("string key path")
	}
	// A path tfviz does not understand withholds the whole value rather
	// than risking exposure.
	unknown := sensitivityMarks([]json.RawMessage{json.RawMessage(`[{"type":"splat"}]`)})
	if buildValue(raw, unknown, nil, nil, nil).Kind() != KindSensitive {
		t.Error("unrecognised steps must mark everything")
	}
}
