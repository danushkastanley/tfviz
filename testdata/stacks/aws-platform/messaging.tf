resource "aws_sqs_queue" "orders" {
  name              = "platform-orders"
  kms_master_key_id = aws_kms_key.platform.arn
}

# Proposed-only: a dead-letter queue for failed orders.
resource "aws_sqs_queue" "orders_dlq" {
  count = local.proposed ? 1 : 0

  name              = "platform-orders-dlq"
  kms_master_key_id = aws_kms_key.platform.arn
}

resource "aws_sns_topic" "events" {
  name              = "platform-events"
  kms_master_key_id = aws_kms_key.platform.arn
}

resource "aws_sns_topic_subscription" "orders" {
  topic_arn = aws_sns_topic.events.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.orders.arn
}

resource "aws_msk_configuration" "platform" {
  name              = "platform"
  kafka_versions    = ["3.9.x"]
  server_properties = "auto.create.topics.enable = false\n"
}

resource "aws_msk_cluster" "platform" {
  cluster_name           = "platform-stream"
  kafka_version          = "3.9.x"
  number_of_broker_nodes = 2

  broker_node_group_info {
    instance_type   = "kafka.m7g.large"
    client_subnets  = local.private_subnet_ids
    security_groups = [aws_security_group.data.id]
  }

  configuration_info {
    arn      = aws_msk_configuration.platform.arn
    revision = aws_msk_configuration.platform.latest_revision
  }

  logging_info {
    broker_logs {
      s3 {
        enabled = true
        bucket  = aws_s3_bucket.logs.id
        prefix  = "msk/"
      }
    }
  }
}
