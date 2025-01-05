# Overview

Welcome to the documentation for GoHead! This guide will help you get started by providing an overview of the key features, structure, and goals of the project.

## Quick Start
To try out GoHead, our Headless CMS written in Go, you can install it and run example workflows or use the provided Docker container for testing.

Alternatively, if you don't want to set up a local environment, you can explore our hosted demo (if available).

### Prerequisites
Before installing GoHead, ensure you have the following:

- A working Go environment (Go version X.Y.Z or higher).
- A database (PostgreSQL, MySQL, or SQLite).
- [Optional] Docker and Docker Compose for containerized setup.

For quick testing, you can use a local setup with:

- Docker Desktop
- Minikube
- Any local Kubernetes solution

### Development vs. Production
These instructions are intended to help you get started quickly. They are not suitable for production. For production setups, please refer to the [Deployment Guide](deployment.md).

## Install GoHead

### Using Go
First, specify the version you want to install in an environment variable. Modify the command below:

```bash
CMS_VERSION="vX.Y.Z"
```

Then, clone the repository and build the project:

```bash
git clone https://github.com/your-org/gohead.git
cd gohead
git checkout $CMS_VERSION
make build
```

### Using Docker
Run the following commands to set up a quick-start environment using Docker:

```bash
docker-compose up -d
```

This will start GoHead along with any required services like a database.

## Submit an Example Workflow

### Using the CLI
You can interact with GoHead using the provided CLI. Submit a sample workflow with the following command:

```bash
cms-cli submit --config config.yaml --watch
```

The `--watch` flag monitors the workflow as it runs and reports its success or failure. When the workflow completes, the watch stops.

To list all submitted workflows:

```bash
cms-cli list
```

You can view the details of a specific workflow using:

```bash
cms-cli get @latest
```

The `@latest` argument is a shortcut to view the most recent workflow.

To observe the logs of the latest workflow:

```bash
cms-cli logs @latest
```

### Using the Web UI
Access the UI for workflow submission:

1. Forward the server's port:
   ```bash
   kubectl port-forward service/cms-server 8080:8080
   ```

2. Navigate your browser to [http://localhost:8080](http://localhost:8080).

3. Click **Submit New Workflow** and provide the necessary configurations.

## Have a Question?
For further assistance, refer to:

- [GitHub Discussions](https://github.com/your-org/gohead/discussions)
- [Slack Channel](https://slack.your-org.com)
