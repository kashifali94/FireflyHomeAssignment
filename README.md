# Savannahtakehomeassi

**Savannahtakehomeassi** is a Go-based project designed to manage and monitor AWS and Terraform configurations. It provides tools for detecting and reporting configuration drift between live AWS resources and their Terraform definitions, ensuring infrastructure remains consistent with the desired state defined in code.

## 📘 Table of Contents
- [Project Overview](#project-overview)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Usage Example](#usage-example)
- [Design Decisions and Trade-offs](#design-decisions-and-trade-offs)
- [Future Improvements](#future-improvements)

---

## 🛠️ Project Overview

This project provides functionality to:
- Compare live AWS EC2 instances with both Terraform state and HCL configurations
- Perform concurrent drift checks against multiple sources
- Implement retry mechanisms for AWS API calls
- Run in a containerized local environment with LocalStack
- Include robust unit testing and modular architecture
- Provide detailed logging of drift detection results

---

## 🧪 Project Structure

The project is organized into several key directories:

- **`awsd/`**: Contains AWS-related modules and configurations
- **`terraform/`**: Houses Terraform-related modules and configurations
- **`configuration/`**: Manages application configuration and environment variables
- **`driftChecker/`**: Contains the core drift detection logic
- **`logger/`**: Handles application logging
- **`main.go`**: The entry point for the application, orchestrating the execution of various modules
- **`Makefile`**: Defines build automation tasks, including test binary creation and Docker image building
- **`Dockerfile`**: Specifies the steps to build a Docker image for the application
- **`docker-compose.yml`**: Defines services and configurations for running the application in a containerized environment

---
## 🚀 Getting Started

### Prerequisites

Ensure you have the following installed:

- [Go 1.18+](https://golang.org/dl/)
- [Docker](https://www.docker.com/get-started)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [Make](https://www.gnu.org/software/make/)

### Building and Running the Application

1. Clone the Repository:
   ```bash
   git clone <repository-url>
   cd Savannahtakehomeassi
   ```

2. Install Dependencies:
   ```bash
   go mod tidy
   ```

3. Build the Application:
   ```bash
   make build
   ```

4. Build Test Binaries:
   ```bash
   make build test-binaries
   ```

5. Start the Environment:
   ```bash
   docker-compose up --build -d
   ```

6. Access the Container:
   ```bash
   docker exec -it drift-checker /bin/sh
   ```

7. Run the Drift Checker:
   ```bash
   ./drift-checker
   ```

8. Run Tests:
   ```bash
   cd test-binaries
   ./<test-binary-name>
   ```

---

### Design Decisions and Trade-offs

- Implemented concurrent drift checking against both Terraform state and HCL configurations
- Added retry mechanism for AWS API calls with configurable attempts and delays
- Leveraged interfaces to enable mock implementations of AWS clients for effective unit testing
- Structured code into domain-specific packages, promoting separation of concerns
- Followed SOLID principles for robust, extensible, and testable architecture
- Adopted a layered and modular design for loose coupling and component reusability
- Utilized environment variables for secure configuration management
- Achieved comprehensive code coverage through unit and integration tests
- Implemented efficient O(n) complexity algorithms for comparing configurations
- Employed LocalStack for local AWS service simulation
- Used structured logging with zap for better observability

---

### Future Improvements

- Add support for multiple AWS regions and resource types
- Implement drift remediation capabilities
- Add web-based dashboard for drift visualization
- Enhance configuration validation and error handling
- Implement drift history tracking and reporting
- Add support for custom drift detection rules
- Improve performance through caching and optimization
- Add support for multiple Terraform workspaces
- Implement drift detection for additional AWS services

---

### Usage

The application continuously monitors for drift between AWS resources and their Terraform definitions. It performs two types of comparisons:

1. AWS vs Terraform State: Compares live AWS instances with the Terraform state file
2. AWS vs HCL Config: Compares live AWS instances with the Terraform HCL configuration

Drift detection results are logged with detailed information about any discrepancies found.

### Sample Configuration

The application uses a `.env` file for configuration. Here's an example:

```env
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
TF_STATE_PATH=/app/tfdata/terraform.tfstate
MAIN_TF_PATH=/app/terraform/main.tf
CHECK_INTERVAL=5m
MAX_RETRIES=3
RETRY_DELAY=5s
LOG_LEVEL=info
```

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `AWS_REGION` | AWS region to use for API calls | `us-east-1` | Yes |
| `AWS_ACCESS_KEY_ID` | AWS access key ID | - | Yes |
| `AWS_SECRET_ACCESS_KEY` | AWS secret access key | - | Yes |
| `TF_STATE_PATH` | Path to the Terraform state file | `/app/tfdata/terraform.tfstate` | Yes |
| `MAIN_TF_PATH` | Path to the main Terraform configuration file | `/app/terraform/main.tf` | Yes |
| `CHECK_INTERVAL` | Interval between drift checks (e.g., "5m", "1h") | `5m` | No |
| `MAX_RETRIES` | Maximum number of retries for AWS API calls | `3` | No |
| `RETRY_DELAY` | Delay between retry attempts (e.g., "5s", "1m") | `5s` | No |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` | No |

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
// also show the drift checker output


