package aws

import (
	p "github.com/danushkastanley/tfviz/internal/projection"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

func registerWorkloads() {
	register("aws_lb", Adapter{
		Family: model.FamilyLoadBalancing, Noun: "Load balancer",
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "load_balancer_type", Label: "Type"},
			{Key: "internal", Label: "Internal"},
			{Key: "dns_name", Label: "DNS name"},
			{Key: "arn", Label: "ARN"},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelSubnetMembership, Field: "subnets", Targets: []string{"aws_subnet"}},
			{Type: model.RelSecurityGroupAttachment, Field: "security_groups", Targets: []string{"aws_security_group"}},
		},
	})
	register("aws_lb_target_group", Adapter{
		Family: model.FamilyLoadBalancing, Noun: "Target group", Placement: vpcScoped,
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "port", Label: "Port"},
			{Key: "protocol", Label: "Protocol"},
			{Key: "target_type", Label: "Target type"},
			tagsWithheld,
		},
	})
	register("aws_lb_listener", Adapter{
		Family: model.FamilyLoadBalancing, Noun: "Listener",
		LabelFrom: []string{"tags.Name"},
		Placement: Placement{Follow: "load_balancer_arn", FollowTargets: []string{"aws_lb"}},
		Fields: []p.Field{
			{Key: "port", Label: "Port"},
			{Key: "protocol", Label: "Protocol"},
			{Key: "ssl_policy", Label: "TLS policy"},
			{Key: "certificate_arn", Label: "Certificate"},
		},
		Relations: []Relation{
			{Type: model.RelServiceReference, Field: "load_balancer_arn", Targets: []string{"aws_lb"}},
			{Type: model.RelServiceReference, Field: "default_action.target_group_arn", Targets: []string{"aws_lb_target_group"}},
			{Type: model.RelServiceReference, Field: "certificate_arn", Targets: []string{"aws_acm_certificate"}},
		},
	})
	register("aws_db_subnet_group", Adapter{
		Family: model.FamilyDatabase, Role: model.RoleAssociation, Noun: "DB subnet group",
		Identity: []string{"name", "id", "arn"},
		Fields:   []p.Field{{Key: "name", Label: "Name"}, {Key: "description", Label: "Description", Withheld: true}},
		Relations: []Relation{
			{Type: model.RelSubnetMembership, Field: "subnet_ids", Targets: []string{"aws_subnet"}},
		},
	})
	register("aws_db_instance", Adapter{
		Family: model.FamilyDatabase, Noun: "Database instance",
		LabelFrom: []string{"identifier", "tags.Name"},
		Identity:  []string{"id", "arn", "identifier"},
		Fields: []p.Field{
			{Key: "identifier", Label: "Identifier"},
			{Key: "engine", Label: "Engine"},
			{Key: "engine_version", Label: "Engine version"},
			{Key: "instance_class", Label: "Instance class"},
			{Key: "allocated_storage", Label: "Storage (GiB)"},
			{Key: "storage_encrypted", Label: "Storage encrypted"},
			{Key: "multi_az", Label: "Multi-AZ"},
			{Key: "publicly_accessible", Label: "Publicly accessible"},
			{Key: "username", Label: "Master username", Withheld: true},
			{Key: "password", Label: "Master password", ChangeOnly: true},
			{Key: "endpoint", Label: "Endpoint"},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelSubnetMembership, Field: "db_subnet_group_name", Targets: []string{"aws_db_subnet_group"}, Through: true},
			{Type: model.RelSecurityGroupAttachment, Field: "vpc_security_group_ids", Targets: []string{"aws_security_group"}},
			{Type: model.RelEncryptionKey, Field: "kms_key_id", Targets: []string{"aws_kms_key"}},
		},
	})
	register("aws_msk_cluster", Adapter{
		Family: model.FamilyStreaming, Noun: "MSK cluster",
		LabelFrom: []string{"cluster_name"},
		Fields: []p.Field{
			{Key: "cluster_name", Label: "Cluster name"},
			{Key: "arn", Label: "ARN"},
			{Key: "kafka_version", Label: "Kafka version"},
			{Key: "number_of_broker_nodes", Label: "Broker count"},
			{Key: "broker_node_group_info.instance_type", Label: "Broker type"},
			{Key: "broker_node_group_info.volume_size", Label: "Storage per broker (GiB)",
				Path: []string{"broker_node_group_info", "storage_info", "ebs_storage_info", "volume_size"}},
			{Key: "encryption_info.client_broker", Label: "Client–broker encryption",
				Path: []string{"encryption_info", "encryption_in_transit", "client_broker"}},
			{Key: "encryption_info.in_cluster", Label: "In-cluster encryption",
				Path: []string{"encryption_info", "encryption_in_transit", "in_cluster"}},
			{Key: "client_authentication.sasl.scram", Label: "SASL/SCRAM"},
			{Key: "client_authentication.sasl.iam", Label: "IAM authentication"},
			{Key: "client_authentication.unauthenticated", Label: "Unauthenticated access"},
			{Key: "logging_info.cloudwatch_log_group", Label: "Broker log group",
				Path: []string{"logging_info", "broker_logs", "cloudwatch_logs", "log_group"}},
			{Key: "logging_info.s3_bucket", Label: "Broker log bucket",
				Path: []string{"logging_info", "broker_logs", "s3", "bucket"}},
			{Key: "configuration_info.revision", Label: "Configuration revision"},
			{Key: "bootstrap_brokers_sasl_scram", Label: "SCRAM bootstrap brokers"},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelSubnetMembership, Field: "broker_node_group_info.client_subnets", Targets: []string{"aws_subnet"}},
			{Type: model.RelSecurityGroupAttachment, Field: "broker_node_group_info.security_groups", Targets: []string{"aws_security_group"}},
			{Type: model.RelEncryptionKey, Field: "encryption_info.encryption_at_rest_kms_key_arn", Targets: []string{"aws_kms_key"}},
			{Type: model.RelLogDestination, Field: "logging_info.broker_logs.cloudwatch_logs.log_group", Targets: []string{"aws_cloudwatch_log_group"}},
			{Type: model.RelLogDestination, Field: "logging_info.broker_logs.s3.bucket", Targets: []string{"aws_s3_bucket"}},
			{Type: model.RelServiceReference, Field: "configuration_info.arn", Targets: []string{"aws_msk_configuration"}},
		},
	})
}
