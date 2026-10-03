package aws

import (
	p "github.com/danushkastanley/tfviz/internal/projection"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// Messaging adapters (Tier B) and MSK configuration.
func registerMessaging() {
	register("aws_sqs_queue", Adapter{
		Family: model.FamilyMessaging, Noun: "SQS queue", Placement: regional,
		Identity: []string{"arn", "id", "url"},
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "fifo_queue", Label: "FIFO"},
			{Key: "visibility_timeout_seconds", Label: "Visibility timeout (s)"},
			{Key: "message_retention_seconds", Label: "Retention (s)"},
			{Key: "policy", Label: "Queue policy", Withheld: true},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelEncryptionKey, Field: "kms_master_key_id", Targets: []string{"aws_kms_key"}},
		},
	})
	register("aws_sns_topic", Adapter{
		Family: model.FamilyMessaging, Noun: "SNS topic", Placement: regional,
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "fifo_topic", Label: "FIFO"},
			{Key: "policy", Label: "Topic policy", Withheld: true},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelEncryptionKey, Field: "kms_master_key_id", Targets: []string{"aws_kms_key"}},
		},
	})
	register("aws_sns_topic_subscription", Adapter{
		Family: model.FamilyMessaging, Role: model.RoleAssociation, Noun: "Subscription",
		Fields: []p.Field{{Key: "protocol", Label: "Protocol"}},
		Relations: []Relation{
			{Type: model.RelServiceReference, From: "topic_arn", FromTargets: []string{"aws_sns_topic"},
				Field: "endpoint", Targets: []string{"aws_sqs_queue", "aws_lambda_function"}},
		},
	})
	// Server properties are free-form broker configuration and are withheld.
	register("aws_msk_configuration", Adapter{
		Family: model.FamilyStreaming, Noun: "MSK configuration", Placement: regional,
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "kafka_versions", Label: "Kafka versions"},
			{Key: "latest_revision", Label: "Latest revision"},
			{Key: "server_properties", Label: "Server properties", Withheld: true},
		},
	})
}
