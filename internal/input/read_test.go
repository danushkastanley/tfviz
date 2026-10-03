package input

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func fixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../testdata/producer/" + path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustRead(t *testing.T, path string, kind SnapshotKind) *Snapshot {
	t.Helper()
	snap, err := Read(fixture(t, path), kind)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return snap
}

func resource(t *testing.T, snap *Snapshot, address string) Resource {
	t.Helper()
	for _, r := range snap.Resources {
		if r.Address == address {
			return r
		}
	}
	t.Fatalf("resource %s not found", address)
	return Resource{}
}

func TestReadsTerraformAndOpenTofuPlans(t *testing.T) {
	tf := mustRead(t, "terraform-1.16/plan.json", KindPlan)
	if tf.Producer != ProducerTerraform || tf.ProducerVersion != "1.16.4" || tf.FormatVersion != "1.2" {
		t.Fatalf("unexpected producer metadata: %+v", tf)
	}
	if tf.Complete == nil || !*tf.Complete || tf.Timestamp == nil {
		t.Fatal("Terraform reports completeness and a timestamp")
	}
	tofu := mustRead(t, "opentofu-1.13/plan.json", KindPlan)
	if tofu.Producer != ProducerOpenTofu || tofu.Complete != nil {
		t.Fatalf("OpenTofu omits completeness; got %+v", tofu.Complete)
	}
	if len(tf.Resources) != 39 || len(tofu.Resources) != 39 {
		t.Fatalf("expected 39 resource changes, got %d and %d", len(tf.Resources), len(tofu.Resources))
	}
	for i := 1; i < len(tf.Resources); i++ {
		if tf.Resources[i-1].Address > tf.Resources[i].Address {
			t.Fatal("resources are not ordered by address")
		}
	}
}

func TestSensitiveValuesCarryOnlyTheirComparison(t *testing.T) {
	snap := mustRead(t, "terraform-1.16/plan.json", KindPlan)
	db := resource(t, snap, "aws_db_instance.orders")
	for name, v := range map[string]Value{"before": db.Before.Field("password"), "after": db.After.Field("password")} {
		if v.Kind() != KindSensitive {
			t.Fatalf("%s password is %v, want sensitive", name, v.Kind())
		}
		if _, ok := v.String(); ok {
			t.Fatalf("%s password exposed a payload", name)
		}
		if v.Comparison() != ComparisonChanged {
			t.Fatalf("%s password comparison = %v, want changed", name, v.Comparison())
		}
	}
	created := resource(t, snap, `module.streaming.aws_secretsmanager_secret_version.orders_scram[0]`)
	if v := created.After.Field("secret_string"); v.Kind() != KindSensitive || v.Comparison() != ComparisonUnavailable {
		t.Fatalf("a newly created secret has no comparison; got kind %v comparison %v", v.Kind(), v.Comparison())
	}
}

func TestMarkedSecretsDoNotSurviveReading(t *testing.T) {
	for _, path := range []string{"terraform-1.16/plan.json", "opentofu-1.13/plan.json", "terraform-1.16/state.json"} {
		kind := KindPlan
		if strings.HasSuffix(path, "state.json") {
			kind = KindState
		}
		snap := mustRead(t, path, kind)
		for _, r := range snap.Resources {
			for _, v := range []Value{r.Before, r.After} {
				walkStrings(v, func(s string) {
					for _, canary := range []string{"CANARY-DB-PASSWORD", "CANARY-KAFKA-PASSWORD"} {
						if strings.Contains(s, canary) {
							t.Fatalf("%s: %s reached the snapshot in %s", path, canary, r.Address)
						}
					}
				})
			}
		}
	}
}

func walkStrings(v Value, visit func(string)) {
	if s, ok := v.String(); ok {
		visit(s)
	}
	for _, item := range v.items {
		walkStrings(item, visit)
	}
	for _, field := range v.fields {
		walkStrings(field, visit)
	}
	visit(v.text)
}

func TestUnknownValuesAreNotNull(t *testing.T) {
	snap := mustRead(t, "terraform-1.16/plan.json", KindPlan)
	nat := resource(t, snap, "aws_nat_gateway.main")
	if nat.After.Field("id").Kind() != KindUnknown {
		t.Fatal("a replaced NAT gateway's new ID is known after apply")
	}
	if got := strings.Join(nat.Actions, ","); got != "delete,create" || nat.ActionReason != "replace_because_cannot_update" {
		t.Fatalf("unexpected actions %q reason %q", got, nat.ActionReason)
	}
	if len(nat.ReplacePaths) != 1 || nat.ReplacePaths[0] != "subnet_id" {
		t.Fatalf("replace paths = %v", nat.ReplacePaths)
	}
}

func TestConfigurationReferencesFollowModuleVariables(t *testing.T) {
	snap := mustRead(t, "terraform-1.16/plan.json", KindPlan)
	msk := resource(t, snap, "module.streaming.aws_msk_cluster.events")
	refs := snap.Config.References(msk, "broker_node_group_info.client_subnets")
	if !contains(refs, `aws_subnet.private["a"].id`) || !contains(refs, `aws_subnet.private["c"].id`) {
		t.Fatalf("client subnets should resolve through var.client_subnet_ids; got %v", refs)
	}
	assoc := resource(t, snap, "module.streaming.aws_msk_scram_secret_association.events[0]")
	refs = snap.Config.References(assoc, "secret_arn_list")
	if !contains(refs, "module.streaming.aws_secretsmanager_secret.orders_scram[0].arn") {
		t.Fatalf("module-local references are made absolute; got %v", refs)
	}
	if msk.ProviderKey != "aws" || snap.Regions["aws"] != "eu-west-1" {
		t.Fatalf("provider key %q, regions %v", msk.ProviderKey, snap.Regions)
	}
}

func TestReadsState(t *testing.T) {
	snap := mustRead(t, "opentofu-1.13/state.json", KindState)
	if snap.Kind != KindState || snap.Producer != ProducerOpenTofu || len(snap.Resources) != 35 {
		t.Fatalf("unexpected state snapshot: kind %s producer %s resources %d", snap.Kind, snap.Producer, len(snap.Resources))
	}
	msk := resource(t, snap, "module.streaming.aws_msk_cluster.events")
	if msk.Module != "module.streaming" || msk.Actions != nil || msk.HasBefore {
		t.Fatalf("state resources carry no change: %+v", msk)
	}
	if resource(t, snap, "aws_db_instance.orders").After.Field("password").Kind() != KindSensitive {
		t.Fatal("state sensitive_values must mark the password")
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func TestRejectsUnsupportedInputWithGuidance(t *testing.T) {
	deep := strings.Repeat("[", MaxDepth+1) + strings.Repeat("]", MaxDepth+1)
	tests := []struct {
		name string
		data string
		kind SnapshotKind
		code Code
	}{
		{"raw state", string(fixture(t, "terraform-1.16/prior.tfstate")), KindState, CodeRawState},
		{"streaming UI", "{\"@level\":\"info\",\"@message\":\"Terraform 1.16.4\",\"type\":\"version\"}\n{\"@level\":\"info\",\"@message\":\"x\"}\n", KindPlan, CodeStreamingUI},
		{"plan to state", string(fixture(t, "terraform-1.16/plan.json")), KindState, CodeWrongKind},
		{"state to plan", string(fixture(t, "terraform-1.16/state.json")), KindPlan, CodeWrongKind},
		{"old format", `{"format_version":"0.2","resource_changes":[]}`, KindPlan, CodeUnsupported},
		{"encrypted", `{"meta":{},"encrypted_data":"abc","encryption_version":"v0"}`, KindState, CodeEncrypted},
		{"not an export", `{"hello":"world"}`, KindPlan, CodeUnsupported},
		{"not JSON", `resource "aws_vpc" "x" {}`, KindPlan, CodeMalformed},
		{"two documents", `{"format_version":"1.2","resource_changes":[]} {}`, KindPlan, CodeMalformed},
		{"too deep", deep, KindPlan, CodeTooDeep},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Read([]byte(tt.data), tt.kind)
			var inputErr *Error
			if !errors.As(err, &inputErr) || inputErr.Code != tt.code {
				t.Fatalf("got %v, want code %s", err, tt.code)
			}
		})
	}
}

func TestMessagesNeverEchoInput(t *testing.T) {
	for _, doc := range []string{
		`{"format_version":"CANARY-SECRET-9","resource_changes":[]}`,
		`{"format_version":"1.2","resource_changes":[{"address":"x","change":{"actions":"CANARY-SECRET-9"}}]}`,
		`CANARY-SECRET-9`,
	} {
		_, err := Read([]byte(doc), KindPlan)
		if err == nil {
			t.Fatalf("accepted %s", doc)
		}
		if strings.Contains(err.Error(), "CANARY") {
			t.Fatalf("error echoed input: %v", err)
		}
	}
}

func TestNewerMinorFormatIsReadWithNotice(t *testing.T) {
	snap, err := Read([]byte(`{"format_version":"1.9","resource_changes":[]}`), KindPlan)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Notices) != 1 || snap.Notices[0].Code != NoticeUnknownFormatMinor {
		t.Fatalf("notices = %+v", snap.Notices)
	}
}

func TestDeletedResourcesHaveNoAfterValue(t *testing.T) {
	snap := mustRead(t, "terraform-1.16/plan.json", KindPlan)
	r := resource(t, snap, "aws_vpc_security_group_ingress_rule.msk_plaintext[0]")
	if r.HasAfter || !r.HasBefore {
		t.Fatalf("a deleted resource has only a before value: before=%v after=%v", r.HasBefore, r.HasAfter)
	}
}
