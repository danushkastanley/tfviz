variable "proposed" {
  type = bool
}

variable "client_subnet_ids" {
  type = list(string)
}

variable "security_group_ids" {
  type = list(string)
}

variable "kafka_password" {
  type      = string
  sensitive = true
}

resource "aws_kms_key" "msk" {
  description             = "MSK at-rest and SCRAM secret encryption"
  deletion_window_in_days = 7
}

resource "aws_cloudwatch_log_group" "broker" {
  name              = "/review/msk/broker"
  retention_in_days = 14
}

# Proposed change: a larger broker type is an in-place update.
resource "aws_msk_cluster" "events" {
  cluster_name           = "review-events"
  kafka_version          = "3.9.x"
  number_of_broker_nodes = 3

  broker_node_group_info {
    instance_type   = var.proposed ? "kafka.m7g.xlarge" : "kafka.m7g.large"
    client_subnets  = var.client_subnet_ids
    security_groups = var.security_group_ids

    storage_info {
      ebs_storage_info {
        volume_size = 500
      }
    }
  }

  encryption_info {
    encryption_at_rest_kms_key_arn = aws_kms_key.msk.arn

    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
  }

  client_authentication {
    sasl {
      scram = true
    }
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.broker.name
      }
    }
  }
}

# Proposed-only: a new SCRAM credential for the orders service.
resource "aws_secretsmanager_secret" "orders_scram" {
  count = var.proposed ? 1 : 0

  name       = "AmazonMSK_review_orders"
  kms_key_id = aws_kms_key.msk.key_id
}

resource "aws_secretsmanager_secret_version" "orders_scram" {
  count = var.proposed ? 1 : 0

  secret_id = aws_secretsmanager_secret.orders_scram[0].id
  secret_string = jsonencode({
    username = "orders"
    password = var.kafka_password
  })
}

resource "aws_msk_scram_secret_association" "events" {
  count = var.proposed ? 1 : 0

  cluster_arn     = aws_msk_cluster.events.arn
  secret_arn_list = [aws_secretsmanager_secret.orders_scram[0].arn]

  depends_on = [aws_secretsmanager_secret_version.orders_scram]
}
