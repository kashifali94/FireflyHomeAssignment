provider "aws" {
  access_key = "test"
  secret_key = "test"
  region     = "us-east-1"

  endpoints {
    ec2 = "http://localhost:4566"
  }

  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
}

resource "aws_instance" "example" {
  ami           = "ami-12345678"  # Should match the one registered in LocalStack
  instance_type = "t2.micro"
#   security_groups = [aws_security_group.test_sg.name]

  tags = {
    Name = "TestInstance"
  }

}

output "instance_id" {
  value = aws_instance.example.id
}

