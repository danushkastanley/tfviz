package graph

import (
	"os"
	"reflect"
	"testing"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/model"
	"github.com/danushkastanley/tfviz/internal/testutil"
)

func build(t *testing.T, path string, kind input.SnapshotKind) *Graph {
	t.Helper()
	data, err := os.ReadFile(testutil.RepoPath("testdata/producer/" + path))
	if err != nil {
		t.Fatal(err)
	}
	snap, err := input.Read(data, kind)
	if err != nil {
		t.Fatal(err)
	}
	return Build(snap)
}

func node(t *testing.T, g *Graph, address string) *Node {
	t.Helper()
	for _, n := range g.Nodes {
		if n.Resource.Address == address {
			return n
		}
	}
	t.Fatalf("no node %s", address)
	return nil
}

type relSummary struct {
	target   string
	typ      model.RelationshipType
	presence model.Presence
	evidence model.EvidenceKind
	via      string
}

func relationshipsFrom(t *testing.T, g *Graph, address string) []relSummary {
	t.Helper()
	src := node(t, g, address)
	addr := map[string]string{}
	for _, n := range g.Nodes {
		addr[n.ID] = n.Resource.Address
	}
	var out []relSummary
	for _, r := range g.Relationships {
		if r.Source == src.ID {
			out = append(out, relSummary{addr[r.Target], r.Type, r.Presence, r.Evidence.Kind, addr[r.Evidence.Via]})
		}
	}
	return out
}

func has(rels []relSummary, want relSummary) bool {
	for _, r := range rels {
		if r == want {
			return true
		}
	}
	return false
}

func group(g *Graph, id string) model.Group {
	for _, grp := range g.Groups {
		if grp.ID == id {
			return grp
		}
	}
	return model.Group{}
}

func TestChangeClassification(t *testing.T) {
	g := build(t, "terraform-1.16/plan.json", input.KindPlan)
	tests := map[string]model.Change{
		"aws_nat_gateway.main":                                 {Action: model.ActionReplace, ReplaceOrder: model.ReplaceDeleteFirst},
		"aws_security_group.app":                               {Action: model.ActionReplace, ReplaceOrder: model.ReplaceCreateFirst},
		"data.aws_security_group.app_lookup[0]":                {Action: model.ActionRead},
		"aws_vpc_security_group_ingress_rule.msk_plaintext[0]": {Action: model.ActionDelete},
		"aws_vpc.main":                                         {Action: model.ActionNoOp},
	}
	for address, want := range tests {
		got := node(t, g, address).Change
		if got.Action != want.Action || got.ReplaceOrder != want.ReplaceOrder {
			t.Errorf("%s: %s/%s, want %s/%s", address, got.Action, got.ReplaceOrder, want.Action, want.ReplaceOrder)
		}
	}
	if fields := node(t, g, "aws_nat_gateway.main").Change.ReplaceFields; !reflect.DeepEqual(fields, []string{"subnet_id"}) {
		t.Errorf("replace fields = %v", fields)
	}
}

func TestRelationshipsUseEvidenceAndNeverInvent(t *testing.T) {
	g := build(t, "terraform-1.16/plan.json", input.KindPlan)
	nat := relationshipsFrom(t, g, "aws_nat_gateway.main")
	if !has(nat, relSummary{`aws_subnet.public["a"]`, model.RelSubnetMembership, model.PresenceBefore, model.EvidenceAttribute, ""}) ||
		!has(nat, relSummary{`aws_subnet.public["b"]`, model.RelSubnetMembership, model.PresenceAfter, model.EvidenceAttribute, ""}) {
		t.Errorf("NAT gateway move: %+v", nat)
	}
	if !has(relationshipsFrom(t, g, "aws_route_table.private"), relSummary{"aws_nat_gateway.main", model.RelRouting, model.PresenceBoth, model.EvidenceConfiguration, ""}) {
		t.Error("a route to a NAT known only after apply resolves through its configuration reference")
	}
	msk := relationshipsFrom(t, g, "module.streaming.aws_msk_cluster.events")
	if !has(msk, relSummary{"module.streaming.aws_secretsmanager_secret.orders_scram[0]", model.RelSecretAssociation, model.PresenceAfter,
		model.EvidenceConfiguration, "module.streaming.aws_msk_scram_secret_association.events[0]"}) {
		t.Errorf("SCRAM association: %+v", msk)
	}
	db := relationshipsFrom(t, g, "aws_db_instance.orders")
	if !has(db, relSummary{`aws_subnet.private["b"]`, model.RelSubnetMembership, model.PresenceBoth, model.EvidenceAttribute, "aws_db_subnet_group.main"}) {
		t.Errorf("RDS subnets through its DB subnet group: %+v", db)
	}
	if rels := relationshipsFrom(t, g, "data.aws_security_group.app_lookup[0]"); len(rels) != 0 {
		t.Errorf("a data source with unknown inputs has no evidenced relationships: %+v", rels)
	}
	platform := build(t, "terraform-1.16-platform/plan.json", input.KindPlan)
	if rels := relationshipsFrom(t, platform, "aws_iam_role.workloads"); len(rels) != 0 {
		t.Errorf("unsupported resources get no relationships: %+v", rels)
	}
}

func TestOnlyGenuineReferencesAreUnresolved(t *testing.T) {
	g := build(t, "terraform-1.16/plan.json", input.KindPlan)
	if len(g.Unresolved) != 1 || g.Unresolved[0].Field != "certificate_arn" || g.Unresolved[0].Reason != model.UnresolvedExternal {
		t.Fatalf("unresolved = %+v", g.Unresolved)
	}
}

func TestPlacement(t *testing.T) {
	g := build(t, "terraform-1.16/plan.json", input.KindPlan)
	at := func(address string) model.Group { return group(g, g.Placement[node(t, g, address).ID].Architecture) }
	vpc := at("module.streaming.aws_msk_cluster.events")
	if vpc.Kind != model.GroupVPC || vpc.Label != "review-platform" {
		t.Errorf("a multi-subnet cluster sits once in its VPC; got %+v", vpc)
	}
	if nat := at("aws_nat_gateway.main"); nat.Kind != model.GroupSubnet || nat.Label != "public-b · eu-west-1b" {
		t.Errorf("a single-subnet resource sits in its post-change subnet; got %+v", nat)
	}
	for _, address := range []string{"module.streaming.aws_kms_key.msk", "aws_ssm_parameter.feature_flag", "aws_eip.nat"} {
		if grp := at(address); grp.Kind != model.GroupRegionalServices {
			t.Errorf("%s must not be placed inside a VPC; got %+v", address, grp)
		}
	}
	if at("aws_lb_listener.https") != vpc || at("aws_db_subnet_group.main") != vpc {
		t.Error("listeners and DB subnet groups follow what they reference")
	}
	if bastion := at("aws_instance.bastion"); bastion.Kind != model.GroupSubnet || bastion.Label != "public-a · eu-west-1a" {
		t.Errorf("an instance sits in its subnet; got %+v", bastion)
	}
	platform := build(t, "terraform-1.16-platform/plan.json", input.KindPlan)
	role := node(t, platform, "aws_iam_role.workloads")
	if group(platform, platform.Placement[role.ID].Architecture).Kind != model.GroupUnplaced {
		t.Error("unsupported resources have no placement")
	}
	region := group(g, vpc.Parent)
	account := group(g, region.Parent)
	if region.Label != "eu-west-1" || account.Label != "Account 111122223333" {
		t.Errorf("hierarchy: %+v / %+v", region, account)
	}
}

func TestSubnetClassificationNeedsRouteEvidence(t *testing.T) {
	g := build(t, "terraform-1.16/plan.json", input.KindPlan)
	for _, grp := range g.Groups {
		if grp.Kind != model.GroupSubnet {
			continue
		}
		want := model.ClassificationPrivate
		if grp.Label[:6] == "public" {
			want = model.ClassificationPublic
		}
		if grp.Classification != want {
			t.Errorf("%s classified %s, want %s", grp.Label, grp.Classification, want)
		}
	}
}

func TestStateHasNoChangesAndNeutralPresence(t *testing.T) {
	g := build(t, "terraform-1.16/state.json", input.KindState)
	for _, n := range g.Nodes {
		if n.Change.Action != model.ActionNoOp {
			t.Fatalf("%s: %s", n.Resource.Address, n.Change.Action)
		}
	}
	for _, r := range g.Relationships {
		if r.Presence != model.PresenceBoth {
			t.Fatalf("state relationships have no before/after: %+v", r)
		}
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	a := build(t, "terraform-1.16/plan.json", input.KindPlan)
	b := build(t, "terraform-1.16/plan.json", input.KindPlan)
	if !reflect.DeepEqual(a.Relationships, b.Relationships) || !reflect.DeepEqual(a.Groups, b.Groups) || !reflect.DeepEqual(a.Placement, b.Placement) {
		t.Fatal("two builds of the same input differ")
	}
}

func TestSensitiveReferencesSuppressRelationships(t *testing.T) {
	doc := `{"format_version":"1.2","resource_changes":[
	 {"address":"aws_security_group.a","mode":"managed","type":"aws_security_group","name":"a","provider_name":"registry.terraform.io/hashicorp/aws",
	  "change":{"actions":["no-op"],"before":{"id":"sg-1","vpc_id":"vpc-1"},"after":{"id":"sg-1","vpc_id":"vpc-1"},"after_unknown":{},"before_sensitive":{},"after_sensitive":{}}},
	 {"address":"aws_lb.b","mode":"managed","type":"aws_lb","name":"b","provider_name":"registry.terraform.io/hashicorp/aws",
	  "change":{"actions":["no-op"],"before":{"security_groups":["sg-1"]},"after":{"security_groups":["sg-1"]},"after_unknown":{},
	  "before_sensitive":{"security_groups":true},"after_sensitive":{"security_groups":true}}}]}`
	snap, err := input.Read([]byte(doc), input.KindPlan)
	if err != nil {
		t.Fatal(err)
	}
	g := Build(snap)
	if len(g.Relationships) != 0 {
		t.Fatalf("a sensitive reference produced a relationship: %+v", g.Relationships)
	}
	if len(g.Unresolved) != 1 || g.Unresolved[0].Reason != model.UnresolvedSensitive {
		t.Fatalf("unresolved = %+v", g.Unresolved)
	}
}

// Attached security groups belong to the resource's VPC, so they place a
// resource that records no subnets.
func TestSecurityGroupsEvidenceTheVPC(t *testing.T) {
	doc := `{"format_version":"1.2","resource_changes":[
	 {"address":"aws_vpc.main","mode":"managed","type":"aws_vpc","name":"main","provider_name":"registry.terraform.io/hashicorp/aws",
	  "change":{"actions":["no-op"],"before":{"id":"vpc-1"},"after":{"id":"vpc-1"},"after_unknown":{},"before_sensitive":{},"after_sensitive":{}}},
	 {"address":"aws_security_group.db","mode":"managed","type":"aws_security_group","name":"db","provider_name":"registry.terraform.io/hashicorp/aws",
	  "change":{"actions":["no-op"],"before":{"id":"sg-1","vpc_id":"vpc-1"},"after":{"id":"sg-1","vpc_id":"vpc-1"},"after_unknown":{},"before_sensitive":{},"after_sensitive":{}}},
	 {"address":"aws_db_instance.x","mode":"managed","type":"aws_db_instance","name":"x","provider_name":"registry.terraform.io/hashicorp/aws",
	  "change":{"actions":["no-op"],"before":{"id":"db-1","vpc_security_group_ids":["sg-1"]},"after":{"id":"db-1","vpc_security_group_ids":["sg-1"]},"after_unknown":{},"before_sensitive":{},"after_sensitive":{}}}]}`
	snap, err := input.Read([]byte(doc), input.KindPlan)
	if err != nil {
		t.Fatal(err)
	}
	g := Build(snap)
	db := node(t, g, "aws_db_instance.x")
	if grp := group(g, g.Placement[db.ID].Architecture); grp.Kind != model.GroupVPC {
		t.Fatalf("placed in %+v, want the security group's VPC", grp)
	}
	for _, grp := range g.Groups {
		if grp.Kind == model.GroupUnplaced {
			t.Fatal("an empty unplaced group was left in the report")
		}
	}
}
