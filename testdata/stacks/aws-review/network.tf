locals {
  azs = {
    a = { az = "eu-west-1a", public_cidr = "10.40.0.0/24", private_cidr = "10.40.10.0/24" }
    b = { az = "eu-west-1b", public_cidr = "10.40.1.0/24", private_cidr = "10.40.11.0/24" }
    c = { az = "eu-west-1c", public_cidr = "10.40.2.0/24", private_cidr = "10.40.12.0/24" }
  }
}

resource "aws_vpc" "main" {
  cidr_block           = "10.40.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name  = "review-platform"
    Owner = "CANARY-TAG-OWNER-7f3a"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
}

resource "aws_subnet" "public" {
  for_each = local.azs

  vpc_id                  = aws_vpc.main.id
  availability_zone       = each.value.az
  cidr_block              = each.value.public_cidr
  map_public_ip_on_launch = true

  tags = { Name = "public-${each.key}" }
}

resource "aws_subnet" "private" {
  for_each = local.azs

  vpc_id            = aws_vpc.main.id
  availability_zone = each.value.az
  cidr_block        = each.value.private_cidr

  tags = { Name = "private-${each.key}" }
}

resource "aws_eip" "nat" {
  domain = "vpc"
}

# Moving the NAT gateway to another subnet forces a delete-then-create replacement.
resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat.id
  subnet_id     = local.proposed ? aws_subnet.public["b"].id : aws_subnet.public["a"].id
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main.id
  }
}

resource "aws_route_table_association" "public" {
  for_each = local.azs

  subnet_id      = aws_subnet.public[each.key].id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "private" {
  for_each = local.azs

  subnet_id      = aws_subnet.private[each.key].id
  route_table_id = aws_route_table.private.id
}
