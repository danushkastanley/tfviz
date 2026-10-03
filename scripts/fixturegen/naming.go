package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Synthetic identifiers use the AWS documentation account and TEST-NET ranges
// so they can never collide with real infrastructure.
const (
	fixtureAccount = "111122223333"
	fixtureRegion  = "eu-west-1"
)

var idPrefixes = map[string]string{
	"aws_vpc":                             "vpc-",
	"aws_subnet":                          "subnet-",
	"aws_internet_gateway":                "igw-",
	"aws_eip":                             "eipalloc-",
	"aws_nat_gateway":                     "nat-",
	"aws_route_table":                     "rtb-",
	"aws_route_table_association":         "rtbassoc-",
	"aws_security_group":                  "sg-",
	"aws_vpc_security_group_ingress_rule": "sgr-",
	"aws_vpc_security_group_egress_rule":  "sgr-",
	"aws_instance":                        "i-",
	"aws_db_instance":                     "db-",
}

// Resource types whose Terraform ID is their ARN.
var idIsARN = map[string]bool{
	"aws_lb": true, "aws_lb_target_group": true, "aws_lb_listener": true,
	"aws_msk_cluster": true, "aws_secretsmanager_secret": true,
}

// Resource types whose Terraform ID is their configured name.
var idIsName = map[string]string{
	"aws_db_subnet_group":      "name",
	"aws_cloudwatch_log_group": "name",
	"aws_ssm_parameter":        "name",
}

var attrPrefixes = map[string]string{
	"default_network_acl_id":       "acl-",
	"default_route_table_id":       "rtb-",
	"main_route_table_id":          "rtb-",
	"default_security_group_id":    "sg-",
	"dhcp_options_id":              "dopt-",
	"network_interface_id":         "eni-",
	"primary_network_interface_id": "eni-",
	"association_id":               "eipassoc-",
	"security_group_rule_id":       "sgr-",
}

func digest(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func str(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

// synthesise returns a deterministic value for one unknown leaf.
func synthesise(rc resourceChange, after map[string]any, schema schemaBlock, path []pathStep) any {
	name := path[len(path)-1].key
	if !path[len(path)-1].isKey && len(path) > 1 {
		name = path[len(path)-2].key
	}
	seed := digest(rc.Address, fmt.Sprint(path))
	topLevel := len(path) == 1

	switch {
	case topLevel && name == "id":
		return syntheticID(rc, after, seed)
	case topLevel && name == "arn":
		return syntheticARN(rc, after, seed)
	case name == "owner_id" || name == "account_id":
		return fixtureAccount
	case name == "key_id" && rc.Type == "aws_kms_key":
		return str(after, "id")
	case name == "availability_zone_id":
		return "euw1-az" + string("123"[int(seed[0])%3])
	case strings.HasPrefix(name, "bootstrap_brokers"):
		return "b-1.review-events.fixture.invalid:9096,b-2.review-events.fixture.invalid:9096"
	case strings.HasSuffix(name, "dns_name") || name == "endpoint" || name == "address":
		return seed[:8] + ".fixture.invalid"
	case strings.HasSuffix(name, "public_ip"):
		return fmt.Sprintf("203.0.113.%d", int(seed[1])%250+1)
	case strings.HasSuffix(name, "private_ip"):
		return fmt.Sprintf("10.40.%d.%d", int(seed[0])%3, int(seed[1])%250+4)
	}
	if prefix, ok := attrPrefixes[name]; ok {
		return prefix + "0" + seed[:16]
	}
	return zeroForType(attributeType(schema, path))
}

func syntheticID(rc resourceChange, after map[string]any, seed string) string {
	if idIsARN[rc.Type] {
		return syntheticARN(rc, after, seed)
	}
	if field, ok := idIsName[rc.Type]; ok && str(after, field) != "" {
		return str(after, field)
	}
	switch rc.Type {
	case "aws_kms_key":
		return fmt.Sprintf("%s-%s-4%s-8%s-%s", seed[:8], seed[8:12], seed[12:15], seed[15:18], seed[18:30])
	case "aws_secretsmanager_secret_version":
		return str(after, "secret_id") + "|" + seed[:32]
	case "aws_msk_scram_secret_association":
		return str(after, "cluster_arn")
	}
	if prefix, ok := idPrefixes[rc.Type]; ok {
		return prefix + "0" + seed[:16]
	}
	return "fx-" + seed[:17]
}

func syntheticARN(rc resourceChange, after map[string]any, seed string) string {
	id := str(after, "id")
	arn := func(service, resource string) string {
		return fmt.Sprintf("arn:aws:%s:%s:%s:%s", service, fixtureRegion, fixtureAccount, resource)
	}
	switch rc.Type {
	case "aws_vpc":
		return arn("ec2", "vpc/"+id)
	case "aws_subnet":
		return arn("ec2", "subnet/"+id)
	case "aws_internet_gateway":
		return arn("ec2", "internet-gateway/"+id)
	case "aws_eip":
		return arn("ec2", "elastic-ip/"+id)
	case "aws_nat_gateway":
		return arn("ec2", "natgateway/"+id)
	case "aws_route_table":
		return arn("ec2", "route-table/"+id)
	case "aws_security_group":
		return arn("ec2", "security-group/"+id)
	case "aws_vpc_security_group_ingress_rule", "aws_vpc_security_group_egress_rule":
		return arn("ec2", "security-group-rule/"+id)
	case "aws_instance":
		return arn("ec2", "instance/"+id)
	case "aws_lb":
		return arn("elasticloadbalancing", "loadbalancer/app/"+str(after, "name")+"/"+seed[:16])
	case "aws_lb_target_group":
		return arn("elasticloadbalancing", "targetgroup/"+str(after, "name")+"/"+seed[:16])
	case "aws_lb_listener":
		lb := strings.TrimPrefix(str(after, "load_balancer_arn"), arn("elasticloadbalancing", "loadbalancer/"))
		return arn("elasticloadbalancing", "listener/"+lb+"/"+seed[:16])
	case "aws_db_instance":
		return arn("rds", "db:"+str(after, "identifier"))
	case "aws_db_subnet_group":
		return arn("rds", "subgrp:"+str(after, "name"))
	case "aws_kms_key":
		return arn("kms", "key/"+id)
	case "aws_cloudwatch_log_group":
		return arn("logs", "log-group:"+str(after, "name"))
	case "aws_msk_cluster":
		return arn("kafka", "cluster/"+str(after, "cluster_name")+"/"+seed[:8]+"-"+seed[8:12]+"-4"+seed[12:15]+"-8"+seed[15:18]+"-"+seed[18:30]+"-s1")
	case "aws_secretsmanager_secret":
		return arn("secretsmanager", "secret:"+str(after, "name")+"-"+seed[:6])
	case "aws_ssm_parameter":
		return arn("ssm", "parameter"+str(after, "name"))
	}
	return arn("fixture", rc.Type+"/"+seed[:16])
}
