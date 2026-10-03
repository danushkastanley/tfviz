package testutil

import (
	"encoding/json"
	"fmt"
)

// SyntheticPlan builds a `show -json` plan with roughly n resources and about
// three relationships each, for performance tests. Values are fictional.
func SyntheticPlan(n int) []byte {
	type change struct {
		Actions         []string       `json:"actions"`
		Before          map[string]any `json:"before"`
		After           map[string]any `json:"after"`
		AfterUnknown    map[string]any `json:"after_unknown"`
		BeforeSensitive map[string]any `json:"before_sensitive"`
		AfterSensitive  map[string]any `json:"after_sensitive"`
	}
	type rc struct {
		Address      string `json:"address"`
		Mode         string `json:"mode"`
		Type         string `json:"type"`
		Name         string `json:"name"`
		Index        string `json:"index"`
		ProviderName string `json:"provider_name"`
		Change       change `json:"change"`
	}
	add := func(out *[]rc, typ string, i int, values map[string]any) {
		key := fmt.Sprint(i)
		*out = append(*out, rc{
			Address: fmt.Sprintf("%s.r[%q]", typ, key), Mode: "managed", Type: typ, Name: "r", Index: key,
			ProviderName: "registry.terraform.io/hashicorp/aws",
			Change: change{Actions: []string{"no-op"}, Before: values, After: values,
				AfterUnknown: map[string]any{}, BeforeSensitive: map[string]any{}, AfterSensitive: map[string]any{}},
		})
	}
	var out []rc
	add(&out, "aws_vpc", 0, map[string]any{"id": "vpc-0", "arn": "arn:aws:ec2:eu-west-1:111122223333:vpc/vpc-0"})
	subnets := 12
	for i := 0; i < subnets; i++ {
		add(&out, "aws_subnet", i, map[string]any{"id": fmt.Sprintf("subnet-%d", i), "vpc_id": "vpc-0", "availability_zone": fmt.Sprintf("eu-west-1%c", 'a'+i%3)})
	}
	groups := n / 10
	for i := 0; i < groups; i++ {
		add(&out, "aws_security_group", i, map[string]any{"id": fmt.Sprintf("sg-%d", i), "vpc_id": "vpc-0", "name": fmt.Sprintf("sg-%d", i)})
	}
	for i := 0; len(out) < n; i++ {
		subnet := func(k int) string { return fmt.Sprintf("subnet-%d", (i+k)%subnets) }
		sg := func(k int) string { return fmt.Sprintf("sg-%d", (i*7+k)%groups) }
		switch i % 3 {
		case 0:
			add(&out, "aws_lb", i, map[string]any{"arn": fmt.Sprintf("arn:aws:elasticloadbalancing:eu-west-1:111122223333:loadbalancer/app/lb-%d/1", i),
				"name": fmt.Sprintf("lb-%d", i), "subnets": []string{subnet(0), subnet(1)}, "security_groups": []string{sg(0)}})
		case 1:
			add(&out, "aws_instance", i, map[string]any{"id": fmt.Sprintf("i-%d", i), "subnet_id": subnet(0), "vpc_security_group_ids": []string{sg(1), sg(2)}})
		default:
			add(&out, "aws_vpc_security_group_ingress_rule", i, map[string]any{"id": fmt.Sprintf("sgr-%d", i), "security_group_id": sg(0), "referenced_security_group_id": sg(3)})
		}
	}
	data, err := json.Marshal(map[string]any{"format_version": "1.2", "terraform_version": "1.16.4", "complete": true, "resource_changes": out})
	if err != nil {
		panic(err)
	}
	return data
}
