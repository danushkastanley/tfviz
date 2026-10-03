package aws

import (
	p "github.com/danushkastanley/tfviz/internal/projection"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// Compute adapters (Tier B). Relationships to IAM roles are not declared:
// IAM is not interpreted yet, and a reference to an uninterpreted resource
// must not be reported as one outside the report.
func registerCompute() {
	register("aws_instance", Adapter{
		Family: model.FamilyCompute, Noun: "EC2 instance", Placement: Placement{SubnetHome: true},
		Fields: []p.Field{
			{Key: "instance_type", Label: "Instance type"},
			{Key: "ami", Label: "AMI"},
			{Key: "availability_zone", Label: "Availability zone"},
			{Key: "private_ip", Label: "Private IP"},
			{Key: "user_data", Label: "User data", Withheld: true},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelSubnetMembership, Field: "subnet_id", Targets: []string{"aws_subnet"}},
			{Type: model.RelSecurityGroupAttachment, Field: "vpc_security_group_ids", Targets: []string{"aws_security_group"}},
		},
	})
	register("aws_ecs_cluster", Adapter{
		Family: model.FamilyCompute, Noun: "ECS cluster", Placement: regional,
		Identity: []string{"arn", "id", "name"},
		Fields:   []p.Field{{Key: "name", Label: "Name"}, {Key: "arn", Label: "ARN"}, tagsWithheld},
	})
	// Container definitions hold images, commands and environment values;
	// they are withheld whole rather than partially parsed.
	register("aws_ecs_task_definition", Adapter{
		Family: model.FamilyCompute, Noun: "Task definition", Placement: regional,
		LabelFrom: []string{"family"},
		Fields: []p.Field{
			{Key: "family", Label: "Family"},
			{Key: "revision", Label: "Revision"},
			{Key: "cpu", Label: "CPU units"},
			{Key: "memory", Label: "Memory (MiB)"},
			{Key: "network_mode", Label: "Network mode"},
			{Key: "requires_compatibilities", Label: "Compatibilities"},
			{Key: "container_definitions", Label: "Container definitions", Withheld: true},
			tagsWithheld,
		},
	})
	register("aws_ecs_service", Adapter{
		Family: model.FamilyCompute, Noun: "ECS service", Placement: regional,
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "desired_count", Label: "Desired tasks"},
			{Key: "launch_type", Label: "Launch type"},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelServiceReference, Field: "cluster", Targets: []string{"aws_ecs_cluster"}},
			{Type: model.RelServiceReference, Field: "task_definition", Targets: []string{"aws_ecs_task_definition"}},
			{Type: model.RelSubnetMembership, Field: "network_configuration.subnets", Targets: []string{"aws_subnet"}},
			{Type: model.RelSecurityGroupAttachment, Field: "network_configuration.security_groups", Targets: []string{"aws_security_group"}},
			{Type: model.RelServiceReference, Field: "load_balancer.target_group_arn", Targets: []string{"aws_lb_target_group"}},
		},
	})
	register("aws_eks_cluster", Adapter{
		Family: model.FamilyCompute, Noun: "EKS cluster", Placement: regional,
		Identity: []string{"name", "arn", "id"},
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "version", Label: "Kubernetes version"},
			{Key: "endpoint", Label: "API endpoint"},
			{Key: "vpc_config.endpoint_public_access", Label: "Public API endpoint"},
			{Key: "vpc_config.endpoint_private_access", Label: "Private API endpoint"},
			{Key: "certificate_authority", Label: "Certificate authority", Withheld: true},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelSubnetMembership, Field: "vpc_config.subnet_ids", Targets: []string{"aws_subnet"}},
			{Type: model.RelSecurityGroupAttachment, Field: "vpc_config.security_group_ids", Targets: []string{"aws_security_group"}},
			{Type: model.RelEncryptionKey, Field: "encryption_config.provider.key_arn", Targets: []string{"aws_kms_key"}},
		},
	})
	register("aws_eks_node_group", Adapter{
		Family: model.FamilyCompute, Noun: "Node group",
		LabelFrom: []string{"node_group_name"},
		Placement: Placement{Follow: "cluster_name", FollowTargets: []string{"aws_eks_cluster"}},
		Fields: []p.Field{
			{Key: "node_group_name", Label: "Name"},
			{Key: "instance_types", Label: "Instance types"},
			{Key: "scaling_config.desired_size", Label: "Desired nodes"},
			{Key: "scaling_config.min_size", Label: "Minimum nodes"},
			{Key: "scaling_config.max_size", Label: "Maximum nodes"},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelServiceReference, Field: "cluster_name", Targets: []string{"aws_eks_cluster"}},
			{Type: model.RelSubnetMembership, Field: "subnet_ids", Targets: []string{"aws_subnet"}},
		},
	})
	// Environment variables are withheld by the deny guard as well.
	register("aws_lambda_function", Adapter{
		Family: model.FamilyCompute, Noun: "Lambda function", Placement: regional,
		LabelFrom: []string{"function_name"},
		Fields: []p.Field{
			{Key: "function_name", Label: "Name"},
			{Key: "package_type", Label: "Package type"},
			{Key: "runtime", Label: "Runtime"},
			{Key: "memory_size", Label: "Memory (MiB)"},
			{Key: "timeout", Label: "Timeout (s)"},
			{Key: "architectures", Label: "Architectures"},
			{Key: "environment", Label: "Environment variables", Withheld: true},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelSubnetMembership, Field: "vpc_config.subnet_ids", Targets: []string{"aws_subnet"}},
			{Type: model.RelSecurityGroupAttachment, Field: "vpc_config.security_group_ids", Targets: []string{"aws_security_group"}},
			{Type: model.RelLogDestination, Field: "logging_config.log_group", Targets: []string{"aws_cloudwatch_log_group"}},
			{Type: model.RelEncryptionKey, Field: "kms_key_arn", Targets: []string{"aws_kms_key"}},
		},
	})
}
