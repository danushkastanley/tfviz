resource "aws_security_group" "alb" {
  name        = "review-alb"
  description = "Public HTTPS entry"
  vpc_id      = aws_vpc.main.id
}

resource "aws_vpc_security_group_ingress_rule" "alb_https" {
  security_group_id = aws_security_group.alb.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "tcp"
  from_port         = 443
  to_port           = 443
}

# Renaming forces replacement; create_before_destroy makes it create-then-delete.
resource "aws_security_group" "app" {
  name        = local.proposed ? "review-app-v2" : "review-app"
  description = "Application tier CANARY-SG-DESCRIPTION-91c2"
  vpc_id      = aws_vpc.main.id

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "db" {
  name        = "review-db"
  description = "PostgreSQL"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "PostgreSQL from app"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.app.id]
  }
}

resource "aws_security_group" "msk" {
  name        = "review-msk"
  description = "Kafka brokers"
  vpc_id      = aws_vpc.main.id
}

resource "aws_vpc_security_group_ingress_rule" "msk_scram" {
  security_group_id            = aws_security_group.msk.id
  referenced_security_group_id = aws_security_group.app.id
  ip_protocol                  = "tcp"
  from_port                    = 9096
  to_port                      = 9096
}

# Removed in the proposed change: plaintext Kafka access is being retired.
resource "aws_vpc_security_group_ingress_rule" "msk_plaintext" {
  count = local.proposed ? 0 : 1

  security_group_id            = aws_security_group.msk.id
  referenced_security_group_id = aws_security_group.app.id
  ip_protocol                  = "tcp"
  from_port                    = 9092
  to_port                      = 9092
}

# Proposed-only lookup whose input is unknown until apply, producing a deferred read.
data "aws_security_group" "app_lookup" {
  count = local.proposed ? 1 : 0

  id = aws_security_group.app.id
}
