package aws

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/projection"
	"github.com/danushkastanley/tfviz/internal/report/model"
	"github.com/danushkastanley/tfviz/internal/testutil"
)

func read(t *testing.T, path string, kind input.SnapshotKind) *input.Snapshot {
	t.Helper()
	data, err := os.ReadFile(testutil.RepoPath("testdata/producer/" + path))
	if err != nil {
		t.Fatal(err)
	}
	snap, err := input.Read(data, kind)
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func TestNoApprovedFieldIsSecretShaped(t *testing.T) {
	for _, typ := range Types() {
		a, _ := Lookup(typ)
		for _, f := range a.Fields {
			if f.Withheld || f.ChangeOnly {
				continue // never exports a payload
			}
			names := append([]string{f.Key, strings.Join(f.Path, ".")}, f.Inputs...)
			for _, name := range names {
				if projection.Denied(name) {
					t.Errorf("%s approves secret-shaped field %q; mark it Withheld", typ, name)
				}
			}
		}
	}
}

func TestEveryFixtureTypeIsSupportedExceptTheDeliberateGenericOne(t *testing.T) {
	snap := read(t, "terraform-1.16/plan.json", input.KindPlan)
	for _, r := range snap.Resources {
		_, ok := Lookup(r.Type)
		if ok == (r.Type == "aws_instance") {
			t.Errorf("%s: supported = %v", r.Type, ok)
		}
	}
}

func TestProjectionNeverExportsCanaries(t *testing.T) {
	inputs := []struct {
		path string
		kind input.SnapshotKind
	}{
		{"terraform-1.16/plan.json", input.KindPlan},
		{"opentofu-1.13/plan.json", input.KindPlan},
		{"terraform-1.16/state.json", input.KindState},
		{"opentofu-1.13/state.json", input.KindState},
	}
	for _, in := range inputs {
		snap := read(t, in.path, in.kind)
		var all []model.MetadataField
		for _, r := range snap.Resources {
			if a, ok := Lookup(r.Type); ok {
				all = append(all, projection.Project(r, a.Fields, in.kind == input.KindPlan)...)
			}
		}
		data, err := json.Marshal(all)
		if err != nil {
			t.Fatal(err)
		}
		testutil.AssertNoCanaries(t, in.path+" metadata", data)
	}
}

func field(t *testing.T, snap *input.Snapshot, address, key string) model.MetadataField {
	t.Helper()
	for _, r := range snap.Resources {
		if r.Address != address {
			continue
		}
		a, _ := Lookup(r.Type)
		for _, f := range projection.Project(r, a.Fields, snap.Kind == input.KindPlan) {
			if f.Key == key {
				return f
			}
		}
	}
	t.Fatalf("%s %s not projected", address, key)
	return model.MetadataField{}
}

func asJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func TestPlanChangeStatuses(t *testing.T) {
	snap := read(t, "terraform-1.16/plan.json", input.KindPlan)
	tests := []struct {
		address, key string
		status       model.ChangeStatus
		before       string
		after        string
	}{
		{"aws_db_instance.orders", "instance_class", model.ChangeChanged, `{"status":"known","value":"db.t4g.medium"}`, `{"status":"known","value":"db.r7g.large"}`},
		{"aws_db_instance.orders", "password", model.ChangeChanged, `{"status":"sensitive"}`, `{"status":"sensitive"}`},
		{"aws_db_instance.orders", "engine", model.ChangeUnchanged, "null", `{"status":"known","value":"postgres"}`},
		{"module.streaming.aws_msk_cluster.events", "broker_node_group_info.instance_type", model.ChangeChanged, `{"status":"known","value":"kafka.m7g.large"}`, `{"status":"known","value":"kafka.m7g.xlarge"}`},
		{"aws_nat_gateway.main", "id", model.ChangeUnknown, `{"status":"known","value":"nat-0` + "", `{"status":"unknown"}`},
		{"module.streaming.aws_secretsmanager_secret.orders_scram[0]", "name", model.ChangeAdded, "null", `{"status":"known","value":"AmazonMSK_review_orders"}`},
		{"aws_vpc_security_group_ingress_rule.msk_plaintext[0]", "ports", model.ChangeRemoved, `{"status":"known","value":"tcp 9092"}`, "null"},
		{"aws_route_table.private", "routes", model.ChangeUnknown, `{"status":"known","value":["0.0.0.0/0 → nat-0`, `{"status":"unknown"}`},
	}
	for _, tt := range tests {
		f := field(t, snap, tt.address, tt.key)
		if f.ChangeStatus != tt.status {
			t.Errorf("%s %s: status %s, want %s", tt.address, tt.key, f.ChangeStatus, tt.status)
		}
		if got := asJSON(f.Before); !strings.HasPrefix(got, tt.before) {
			t.Errorf("%s %s: before %s, want prefix %s", tt.address, tt.key, got, tt.before)
		}
		if got := asJSON(f.After); got != tt.after {
			t.Errorf("%s %s: after %s, want %s", tt.address, tt.key, got, tt.after)
		}
	}
}

func TestStateProjectionHasNoChangeStatus(t *testing.T) {
	snap := read(t, "terraform-1.16/state.json", input.KindState)
	f := field(t, snap, "module.streaming.aws_msk_cluster.events", "kafka_version")
	if f.ChangeStatus != "" || f.Before != nil || asJSON(f.After) != `{"status":"known","value":"3.9.x"}` {
		t.Fatalf("state field = %s", asJSON(f))
	}
}

func TestFormattersOmitFreeFormText(t *testing.T) {
	snap := read(t, "terraform-1.16/plan.json", input.KindPlan)
	routes := field(t, snap, "aws_route_table.public", "routes")
	if !strings.Contains(asJSON(routes.After), "0.0.0.0/0 → igw-0") {
		t.Fatalf("public routes = %s", asJSON(routes.After))
	}
	ingress := field(t, snap, "aws_security_group.db", "ingress")
	if strings.Contains(asJSON(ingress), "PostgreSQL from app") {
		t.Fatal("inline rule descriptions are free-form and must not be exported")
	}
	if !strings.Contains(asJSON(ingress.Before), "tcp 5432 from sg-") {
		t.Fatalf("ingress = %s", asJSON(ingress))
	}
}
