# Quick Start

This guide will help you quickly get up and running with GoHead using the pre-built Docker image.

## Prerequisites
Before starting, ensure you have the following installed:

- [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/)
- [curl](https://curl.se/) or any HTTP client for testing the API

## Running GoHead with Docker

### Step 1: Pull the Docker Image
Download the latest GoHead Docker image:

```bash
docker pull gohead:latest
```

### Step 2: Create a Configuration File
Create a `config.yaml` file in your current directory with the following contents:

```yaml
log_level: "info"
telemetry_enabled: false
jwt_secret: "your-secret-key"
mode: test
database_url: "sqlite://gohead-local.db"
server_port: "8080"
```

> **Note:** Customize the configuration as needed. You can also use environment variables prefixed with `GOHEAD_` to override these values.

### Step 3: Start the Container
Run the GoHead Docker container with the following command:

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  gohead:latest
```

This command:
- Maps port `8080` of the container to your local port `8080`
- Mounts your local `config.yaml` file into the container

### Step 4: Verify the Server is Running
Check the logs to ensure GoHead is running:

```bash
docker logs $(docker ps -q --filter ancestor=gohead:latest)
```

You should see output indicating that the server has started successfully.

## Testing the API

### Step 1: Access the API
Use `curl` or a browser to access the GoHead API. For example, check the server's health:

```bash
curl http://localhost:8080/healthz
```

You should receive a response like:

```json
{
  "status": "ok"
}
```

### Step 2: Create a Test Resource
Submit a request to create a new resource:

```bash
curl -X POST http://localhost:8080/api/resource \
  -H "Content-Type: application/json" \
  -d '{"name": "test resource"}'
```

You should receive a response confirming the resource creation.

## Stopping the Container
To stop the container, run:

```bash
docker stop $(docker ps -q --filter ancestor=gohead:latest)
```

To remove the container:

```bash
docker rm $(docker ps -aq --filter ancestor=gohead:latest)
```

---
You are now ready to explore GoHead! For advanced setup options, refer to the [Configuration Overview](configuration-overview.md).
