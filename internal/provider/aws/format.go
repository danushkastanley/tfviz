package aws

import (
	"fmt"
	"strings"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/projection"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

func text(v input.Value, path ...string) string {
	s, _ := v.Path(path...).String()
	return s
}

func number(v input.Value, path ...string) string {
	if n, ok := v.Path(path...).Number(); ok {
		return fmt.Sprintf("%g", n)
	}
	return ""
}

// blocks returns the elements of a nested block list (or a single object).
func blocks(v input.Value) []input.Value {
	if items := v.Items(); items != nil {
		return items
	}
	if v.Kind() == input.KindMap {
		return []input.Value{v}
	}
	return nil
}

var routeTargets = []string{"gateway_id", "nat_gateway_id", "transit_gateway_id", "vpc_peering_connection_id",
	"network_interface_id", "egress_only_gateway_id", "vpc_endpoint_id", "local_gateway_id", "carrier_gateway_id", "core_network_arn"}

// formatRoutes lists inline routes as "destination → target".
func formatRoutes(v input.Value) model.FieldValue {
	var out []string
	for _, route := range blocks(v) {
		dest := firstNonEmpty(text(route, "cidr_block"), text(route, "ipv6_cidr_block"), text(route, "destination_prefix_list_id"))
		target := ""
		for _, key := range routeTargets {
			if t := text(route, key); t != "" {
				target = t
				break
			}
		}
		out = append(out, strings.TrimSpace(dest+" → "+firstNonEmpty(target, "unspecified")))
	}
	return projection.List(out)
}

// formatInlineRules summarises inline security group rules without their
// free-form descriptions.
func formatInlineRules(v input.Value) model.FieldValue {
	var out []string
	for _, rule := range blocks(v) {
		sources, _ := rule.Field("cidr_blocks").Strings()
		v6, _ := rule.Field("ipv6_cidr_blocks").Strings()
		groups, _ := rule.Field("security_groups").Strings()
		sources = append(append(sources, v6...), groups...)
		if self, _ := rule.Field("self").Bool(); self {
			sources = append(sources, "self")
		}
		out = append(out, strings.TrimSpace(ports(text(rule, "protocol"), number(rule, "from_port"), number(rule, "to_port"))+" from "+strings.Join(sources, ", ")))
	}
	return projection.List(out)
}

// formatRulePorts renders a standalone rule's protocol and port range.
func formatRulePorts(v input.Value) model.FieldValue {
	return model.KnownString(ports(text(v, "ip_protocol"), number(v, "from_port"), number(v, "to_port")))
}

// formatLegacyRulePorts renders an aws_security_group_rule's protocol and ports.
func formatLegacyRulePorts(v input.Value) model.FieldValue {
	return model.KnownString(ports(text(v, "protocol"), number(v, "from_port"), number(v, "to_port")))
}

func ports(protocol, from, to string) string {
	switch {
	case protocol == "-1" || protocol == "all":
		return "all traffic"
	case from == "" || from == to:
		return strings.TrimSpace(protocol + " " + from)
	default:
		return fmt.Sprintf("%s %s–%s", protocol, from, to)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
