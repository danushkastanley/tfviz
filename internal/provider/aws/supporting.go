package aws

import (
	p "github.com/danushkastanley/tfviz/internal/projection"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

var regional = Placement{Regional: true}

func registerSupporting() {
	register("aws_kms_key", Adapter{
		Family: model.FamilyEncryption, Noun: "KMS key", Placement: regional,
		Identity: []string{"key_id", "arn", "id"},
		Fields: []p.Field{
			{Key: "key_id", Label: "Key ID"},
			{Key: "key_usage", Label: "Usage"},
			{Key: "enable_key_rotation", Label: "Automatic rotation"},
			{Key: "deletion_window_in_days", Label: "Deletion window (days)"},
			{Key: "description", Label: "Description", Withheld: true},
			{Key: "policy", Label: "Key policy", Withheld: true},
			tagsWithheld,
		},
	})
	register("aws_cloudwatch_log_group", Adapter{
		Family: model.FamilyObservability, Noun: "Log group", Placement: regional,
		Identity: []string{"name", "arn", "id"},
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "retention_in_days", Label: "Retention (days)"},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelEncryptionKey, Field: "kms_key_id", Targets: []string{"aws_kms_key"}},
		},
	})
	// Secrets are reference-only: names, ARNs and keys, never their contents.
	register("aws_secretsmanager_secret", Adapter{
		Family: model.FamilySecrets, Noun: "Secret", Placement: regional,
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "arn", Label: "ARN"},
			{Key: "recovery_window_in_days", Label: "Recovery window (days)"},
			{Key: "description", Label: "Description", Withheld: true},
			{Key: "policy", Label: "Resource policy", Withheld: true},
			tagsWithheld,
		},
		Relations: []Relation{
			{Type: model.RelEncryptionKey, Field: "kms_key_id", Targets: []string{"aws_kms_key"}},
		},
	})
	register("aws_secretsmanager_secret_version", Adapter{
		Family: model.FamilySecrets, Noun: "Secret value", Placement: regional,
		Fields: []p.Field{
			{Key: "secret_string", Label: "Secret value", Withheld: true},
			{Key: "secret_binary", Label: "Secret binary", Withheld: true},
		},
		Relations: []Relation{
			{Type: model.RelServiceReference, Field: "secret_id", Targets: []string{"aws_secretsmanager_secret"}},
		},
	})
	register("aws_msk_scram_secret_association", Adapter{
		Family: model.FamilyStreaming, Role: model.RoleAssociation, Noun: "SCRAM secret association",
		Relations: []Relation{
			{Type: model.RelSecretAssociation, From: "cluster_arn", FromTargets: []string{"aws_msk_cluster"},
				Field: "secret_arn_list", Targets: []string{"aws_secretsmanager_secret"}},
		},
	})
	register("aws_ssm_parameter", Adapter{
		Family: model.FamilyConfiguration, Noun: "Parameter", Placement: regional,
		Fields: []p.Field{
			{Key: "name", Label: "Name"},
			{Key: "type", Label: "Type"},
			{Key: "tier", Label: "Tier"},
			{Key: "value", Label: "Value", Withheld: true},
			{Key: "insecure_value", Label: "Value", Withheld: true},
			{Key: "description", Label: "Description", Withheld: true},
			tagsWithheld,
		},
	})
}
