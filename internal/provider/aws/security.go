package aws

import (
	p "github.com/danushkastanley/tfviz/internal/projection"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

func registerSecurity() {
	// Descriptions are free-form text and are withheld by default.
	register("aws_security_group", Adapter{
		Family: model.FamilySecurity, Noun: "Security group", Placement: vpcScoped,
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "id", Label: "Security group ID"},
			{Key: "ingress", Label: "Ingress", Format: formatInlineRules},
			{Key: "egress", Label: "Egress", Format: formatInlineRules},
			{Key: "description", Label: "Description", Withheld: true},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelSecurityGroupRule, Field: "ingress.security_groups", Targets: []string{"aws_security_group"}},
		},
	})
	rule := Adapter{
		Family: model.FamilySecurity, Role: model.RoleAssociation, Noun: "Security group rule",
		Fields: []p.Field{
			{Key: "ports", Label: "Ports", Format: formatRulePorts, Inputs: []string{"ip_protocol", "from_port", "to_port"}},
			{Key: "cidr_ipv4", Label: "Source or destination IPv4"},
			{Key: "cidr_ipv6", Label: "Source or destination IPv6"},
			{Key: "referenced_security_group_id", Label: "Referenced security group"},
			{Key: "description", Label: "Description", Withheld: true},
		},
		Relations: []Relation{
			{Type: model.RelSecurityGroupRule, From: "security_group_id", FromTargets: []string{"aws_security_group"},
				Field: "referenced_security_group_id", Targets: []string{"aws_security_group"}},
		},
	}
	register("aws_vpc_security_group_ingress_rule", rule)
	register("aws_vpc_security_group_egress_rule", rule)
	register("aws_security_group_rule", Adapter{
		Family: model.FamilySecurity, Role: model.RoleAssociation, Noun: "Security group rule",
		Fields: []p.Field{
			{Key: "type", Label: "Direction"},
			{Key: "ports", Label: "Ports", Format: formatLegacyRulePorts, Inputs: []string{"protocol", "from_port", "to_port"}},
			{Key: "cidr_blocks", Label: "CIDR blocks"},
			{Key: "description", Label: "Description", Withheld: true},
		},
		Relations: []Relation{
			{Type: model.RelSecurityGroupRule, From: "security_group_id", FromTargets: []string{"aws_security_group"},
				Field: "source_security_group_id", Targets: []string{"aws_security_group"}},
		},
	})
}
