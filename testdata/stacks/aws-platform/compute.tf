resource "aws_instance" "worker" {
  ami                    = "ami-0fixture000000002"
  instance_type          = local.proposed ? "m7g.large" : "m7g.medium"
  subnet_id              = aws_subnet.private["a"].id
  vpc_security_group_ids = [aws_security_group.workloads.id]
  user_data              = "#!/bin/sh\nexport API_TOKEN=CANARY-USERDATA-TOKEN-5d1f\n"
  tags                   = { Name = "platform-worker" }
}

resource "aws_ecs_cluster" "platform" {
  name = "platform"
}

resource "aws_ecs_task_definition" "api" {
  family                   = "platform-api"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "512"
  memory                   = "1024"
  execution_role_arn       = aws_iam_role.workloads.arn
  container_definitions = jsonencode([{
    name         = "api"
    image        = "111122223333.dkr.ecr.eu-west-1.amazonaws.com/api:1.4.2"
    essential    = true
    environment  = [{ name = "DATABASE_URL", value = "postgres://api:CANARY-ECS-ENV-63aa@db.internal/api" }]
    portMappings = [{ containerPort = 8080 }]
  }])
}

resource "aws_ecs_service" "api" {
  name            = "platform-api"
  cluster         = aws_ecs_cluster.platform.arn
  task_definition = aws_ecs_task_definition.api.arn
  desired_count   = local.proposed ? 4 : 2
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = local.private_subnet_ids
    security_groups = [aws_security_group.workloads.id]
  }
}

resource "aws_eks_cluster" "platform" {
  name     = "platform"
  role_arn = aws_iam_role.workloads.arn
  version  = "1.34"

  vpc_config {
    subnet_ids         = local.private_subnet_ids
    security_group_ids = [aws_security_group.workloads.id]
  }
}

resource "aws_eks_node_group" "general" {
  cluster_name    = aws_eks_cluster.platform.name
  node_group_name = "general"
  node_role_arn   = aws_iam_role.workloads.arn
  subnet_ids      = local.private_subnet_ids
  instance_types  = ["m7g.large"]

  scaling_config {
    desired_size = 3
    max_size     = 6
    min_size     = 3
  }
}

resource "aws_lambda_function" "ingest" {
  function_name = "platform-ingest"
  role          = aws_iam_role.workloads.arn
  package_type  = "Image"
  image_uri     = "111122223333.dkr.ecr.eu-west-1.amazonaws.com/ingest:2.0.0"
  memory_size   = local.proposed ? 1024 : 512
  timeout       = 30

  environment {
    variables = { API_KEY = "CANARY-LAMBDA-ENV-0c4d" }
  }

  vpc_config {
    subnet_ids         = local.private_subnet_ids
    security_group_ids = [aws_security_group.workloads.id]
  }

  logging_config {
    log_format = "JSON"
    log_group  = aws_cloudwatch_log_group.apps.name
  }
}
