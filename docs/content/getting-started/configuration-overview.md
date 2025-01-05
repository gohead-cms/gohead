# Configuration Overview

This document provides an overview of the configuration file for GoHead. The `config.yaml` file allows you to customize the behavior of the system by setting various parameters. Additionally, you can use environment variables prefixed with `GOHEAD_` to override these values.

## Sample Configuration File
Below is a sample `config.yaml` file:

```yaml
# config.yaml
log_level: "info"
telemetry_enabled: false
jwt_secret: "your-secret-key"
mode: test
# database_url: "sqlite://gohead-local.db"
# database_url: "mysql://gohead_user:gohead_pass@tcp(127.0.0.1:3306)/gohead?charset=utf8mb4&parseTime=True&loc=Local"
database_url: "postgres://gohead_user:pass@localhost:5432/gohead?sslmode=disable"
server_port: "8080"
```

## Configuration Parameters

### General Settings
- **`log_level`**: Sets the logging verbosity level. Possible values are `debug`, `info`, `warn`, `error`. Default is `info`.
- **`telemetry_enabled`**: Enables or disables telemetry data collection. Default is `false`.
- **`jwt_secret`**: Secret key used for JWT authentication. Replace this with a strong secret in production environments.
- **`mode`**: Determines the operating mode of GoHead. Common values include `test` and `production`.

### Database Configuration
- **`database_url`**: Connection string for the database. Supported databases include:
  - SQLite
  - MySQL
  - PostgreSQL

  Examples:
  - SQLite: `sqlite://gohead-local.db`
  - MySQL: `mysql://gohead_user:gohead_pass@tcp(127.0.0.1:3306)/gohead?charset=utf8mb4&parseTime=True&loc=Local`
  - PostgreSQL: `postgres://gohead_user:pass@localhost:5432/gohead?sslmode=disable`

### Server Configuration
- **`server_port`**: Specifies the port on which the server runs. Default is `8080`.

## Using Environment Variables
You can override configuration settings using environment variables prefixed with `GOHEAD_`. For example:

- To override `log_level`:
  ```bash
  export GOHEAD_LOG_LEVEL=debug
  ```

- To override `server_port`:
  ```bash
  export GOHEAD_SERVER_PORT=9090
  ```

Environment variable names should match the configuration keys but in uppercase, with underscores replacing periods.

## Tips for Production
- Always set `jwt_secret` to a secure, random value.
- Use a database system suitable for your scale (e.g., PostgreSQL for production).
- Review all configuration parameters and test your setup before deploying.

For more details on advanced configuration and deployment, refer to the [Deployment Guide](deployment.md).
