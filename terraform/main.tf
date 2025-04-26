provider "aws" {
  access_key               = "test"
  secret_key               = "test"
  region                   = "us-east-1"
  endpoints {
    ec2 = "http://localhost:4566"
  }
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
}

resource "aws_instance" "example" {
  ami           = "ami-123456789"  # Should match the one registered in LocalStack
  instance_type = "t3.micro"
  tags = {
    Name = "TestInstance"
  }
}

output "instance_id" {
  value = aws_instance.example.id
}

terraform {
  backend "local" {
    path = "tfdata/terraform.tfstate"
  }
}
