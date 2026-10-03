resource "aws_lb" "public" {
  name               = "review-public"
  load_balancer_type = "application"
  internal           = false
  security_groups    = [aws_security_group.alb.id]
  subnets            = [aws_subnet.public["a"].id, aws_subnet.public["b"].id, aws_subnet.public["c"].id]
}

resource "aws_lb_target_group" "app" {
  name        = "review-app"
  port        = 8080
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = aws_vpc.main.id
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.public.arn
  port              = 443
  protocol          = "HTTPS"
  certificate_arn   = "arn:aws:acm:eu-west-1:111122223333:certificate/00000000-0000-4000-8000-000000000001"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app.arn
  }
}

resource "aws_db_subnet_group" "main" {
  name       = "review-db"
  subnet_ids = [aws_subnet.private["a"].id, aws_subnet.private["b"].id, aws_subnet.private["c"].id]
}

# Proposed change: larger instance class (in-place update) and a rotated password.
resource "aws_db_instance" "orders" {
  identifier             = "review-orders"
  engine                 = "postgres"
  engine_version         = "17.4"
  instance_class         = local.proposed ? "db.r7g.large" : "db.t4g.medium"
  allocated_storage      = 50
  storage_encrypted      = true
  username               = "orders_admin"
  password               = local.proposed ? "${var.db_password}-rotated" : var.db_password
  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.db.id]
  skip_final_snapshot    = true
}

# Unmarked secret material: insecure_value is not flagged sensitive by the provider.
resource "aws_ssm_parameter" "feature_flag" {
  name           = "/review/feature-flag"
  type           = "String"
  insecure_value = "CANARY-SSM-INSECURE-VALUE-2b8e"
}

# Unsupported in the first slice: exercises the limited generic card and user_data handling.
resource "aws_instance" "bastion" {
  ami                    = "ami-0fixture000000001"
  instance_type          = "t4g.micro"
  subnet_id              = aws_subnet.public["a"].id
  vpc_security_group_ids = [aws_security_group.app.id]
  user_data              = "#!/bin/sh\nexport API_TOKEN=CANARY-USERDATA-TOKEN-5d1f\n"
}

module "streaming" {
  source = "./modules/streaming"

  proposed           = local.proposed
  client_subnet_ids  = [aws_subnet.private["a"].id, aws_subnet.private["b"].id, aws_subnet.private["c"].id]
  security_group_ids = [aws_security_group.msk.id]
  kafka_password     = var.kafka_password
}
