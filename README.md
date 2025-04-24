# Savannahtakehomeassi

**Savannahtakehomeassi** is a Go-based project designed to manage and monitor AWS and Terraform configurations. It aims to provide tools for detecting and reporting configuration drift, ensuring that infrastructure remains consistent with the desired state defined in code.

## 🧪 Project Structure

The project is organized into several key directories:

- **`awsd/`**: Contains AWS-related modules and configurations.
- **`terraform/`**: Houses Terraform-related modules and configurations.
- **`main.go`**: The entry point for the application, orchestrating the execution of various modules.
- **`Makefile`**: Defines build automation tasks, including test binary creation and Docker image building.
- **`Dockerfile`**: Specifies the steps to build a Docker image for the application.
- **`docker-compose.yml`**: Defines services and configurations for running the application in a containerized environment.

## 🚀 Getting Started

### Prerequisites

Ensure you have the following installed:

- [Go 1.18+](https://golang.org/dl/)
- [Docker](https://www.docker.com/get-started)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [Make](https://www.gnu.org/software/make/)

### Building the Application

To build the application and its test binaries:

### Step to run and installation
- Clone the Repo
- **Cd** into **Savannahtakehomeassi** Directory
- Run **go mod tidy** if you see any dependency related issue. 
- Run **make build** command to build the go binary
- Run **make build test-binaries** command to build the test cases binaries
- Run **docker-compose up --build -d** command to up the enviorment. It will create images and appropiate volumes
- Once everything is running. Run **docker exec -it drift-checker /bin/sh** to get into the container.
- Wait for few seconds to Run **./drift-checker** command inside the drift  checker container. Terraform takes time few seconds to create the instance.
- **Cd** into to the **test-binaries** directory. Now run test binary by adding **./** before binary you will see the coverage of the test cases.

### Design decisions and trade-offs
- Leveraged interfaces to enable mock implementations of AWS clients for effective and isolated unit testing.
- Structured code into domain-specific packages, promoting separation of concerns and improved maintainability.
- Followed SOLID principles, ensuring a robust, extensible, and testable architecture.
- Adopted a layered and modular design to enable loose coupling and simplify component reusability.
- Utilized a .env file for managing environment variables, making configuration more portable and secure.
- Achieved approximately 80% code coverage, reinforcing confidence in code reliability.
- Implemented efficient O(n) complexity algorithms for comparing slices (e.g., structs, arrays, strings), optimizing performance.
- Employed LocalStack to simulate AWS services in a local development environment for faster and safer testing.
- Chose struct-based JSON parsing over dynamic map[string]interface{} to improve type safety, readability, and reduce runtime errors.

### Future improvments
- Refactor and modularize existing code by introducing additional helper methods to improve clarity and maintainability.
- Consolidate the testing process by generating a unified test binary that covers the entire project.
- Integrate AWS resource provisioning within the same container environment, streamlining local setup with LocalStack.
- Expand the test suite with additional unit and integration tests to enhance coverage and ensure reliability across edge cases.

  

### Sample input Terraform configuration
```hcl
provider "aws" {
  access_key = "test"
  secret_key = "test"
  region     = "us-east-1"

  endpoints {
    ec2 = "http://localstack:4566"
  }

  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
}

resource "aws_instance" "example" {
  ami           = "ami-12345678"  # Should match the one registered in LocalStack
  instance_type = "t2.micro"

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
```


### Sample AWS EC2 response (or mock data)
```json
{
    "Reservations": [
        {
            "ReservationId": "r-71a5ce4fcf3cac876",
            "OwnerId": "000000000000",
            "Groups": [],
            "Instances": [
                {
                    "Architecture": "x86_64",
                    "BlockDeviceMappings": [
                        {
                            "DeviceName": "/dev/sda1",
                            "Ebs": {
                                "AttachTime": "2025-04-24T00:47:55+00:00",
                                "DeleteOnTermination": true,
                                "Status": "in-use",
                                "VolumeId": "vol-613c8e77fa240a928"
                            }
                        }
                    ],
                    "ClientToken": "ABCDE0000000000003",
                    "EbsOptimized": false,
                    "Hypervisor": "xen",
                    "NetworkInterfaces": [
                        {
                            "Association": {
                                "IpOwnerId": "000000000000",
                                "PublicIp": "54.214.76.84"
                            },
                            "Attachment": {
                                "AttachTime": "2015-01-01T00:00:00+00:00",
                                "AttachmentId": "eni-attach-b91106693349f7f16",
                                "DeleteOnTermination": true,
                                "DeviceIndex": 0,
                                "Status": "attached"
                            },
                            "Description": "Primary network interface",
                            "Groups": [
                                {
                                    "GroupId": "sg-9b3820dc6787bb7c1",
                                    "GroupName": "default"
                                }
                            ],
                            "MacAddress": "1b:2b:3c:4d:5e:6f",
                            "NetworkInterfaceId": "eni-66c2ef06dc5b5f075",
                            "OwnerId": "000000000000",
                            "PrivateIpAddress": "10.168.130.93",
                            "PrivateIpAddresses": [
                                {
                                    "Association": {
                                        "IpOwnerId": "000000000000",
                                        "PublicIp": "54.214.76.84"
                                    },
                                    "Primary": true,
                                    "PrivateIpAddress": "10.168.130.93"
                                }
                            ],
                            "SourceDestCheck": true,
                            "Status": "in-use",
                            "SubnetId": "subnet-19c7a2ce20b3c312f",
                            "VpcId": "vpc-a60d11f67460d4c64"
                        }
                    ],
                    "RootDeviceName": "/dev/sda1",
                    "RootDeviceType": "ebs",
                    "SecurityGroups": [],
                    "SourceDestCheck": true,
                    "StateReason": {
                        "Code": "",
                        "Message": ""
                    },
                    "VirtualizationType": "paravirtual",
                    "InstanceId": "i-c83ea05889deacc7a",
                    "ImageId": "ami-12345678",
                    "State": {
                        "Code": 16,
                        "Name": "running"
                    },
                    "PrivateDnsName": "ip-10-168-130-93.ec2.internal",
                    "PublicDnsName": "ec2-54-214-76-84.compute-1.amazonaws.com",
                    "StateTransitionReason": "",
                    "AmiLaunchIndex": 0,
                    "InstanceType": "t2.micro",
                    "LaunchTime": "2025-04-24T00:47:55+00:00",
                    "Placement": {
                        "GroupName": "",
                        "Tenancy": "default",
                        "AvailabilityZone": "us-east-1a"
                    },
                    "KernelId": "None",
                    "Monitoring": {
                        "State": "disabled"
                    },
                    "SubnetId": "subnet-19c7a2ce20b3c312f",
                    "VpcId": "vpc-a60d11f67460d4c64",
                    "PrivateIpAddress": "10.168.130.93",
                    "PublicIpAddress": "54.214.76.84"
                }
            ]
        }
    ]
}
```



