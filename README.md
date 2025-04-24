cat > README.md <<'EOF'
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

### Step to run
- Clone the Repo
- Cd into Savannahtakehomeassi Directory
- Run make build command to build the go binary
- Run make build test-binaries command to build the test cases binaries
- Run docker-compose up --build -d command to up the enviorment. It will create images and appropiate volumes
- Once everything is running. Run docker exec -it drift-checker /bin/sh to get into the container.
- Wait for few seconds to Run ./drift-checker command inside the drift  checker container. Terraform takes time few seconds to create the instance.
- CD into to the test-binaries directory. Now run test binary by adding ./ before binary you  will see the coverage of the test cases.

