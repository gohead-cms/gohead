# Frequently Asked Questions (FAQ)

Here are answers to some of the most common questions about GoHead.

## General Questions

### What is GoHead?
GoHead is a Headless CMS written in Go, designed for flexibility, performance, and ease of integration with modern web and mobile applications.

### How do I get started with GoHead?
Check out the [Getting Started](getting-started/overview.md) guide for installation and setup instructions.

### Is GoHead open source?
Yes, GoHead is open source. You can find the source code on our [GitHub repository](https://github.com/your-org/gohead).

## Installation and Configuration

### What are the prerequisites for using GoHead?
You need:
- A working Go environment (Go version X.Y.Z or higher).
- A database (PostgreSQL, MySQL, or SQLite).
- [Optional] Docker and Docker Compose for containerized setup.

### Can I use environment variables for configuration?
Yes, all configuration options in the `config.yaml` file can be overridden using environment variables prefixed with `GOHEAD_`. For more details, see the [Configuration Overview](getting-started/configuration-overview.md).

### How do I run GoHead in production?
Refer to the [Deployment Guide](deployment.md) for best practices and production setup instructions.

## Troubleshooting

### I’m having trouble connecting to the database. What should I do?
Ensure that:
- The `database_url` in your configuration is correct.
- Your database service is running and accessible.
- You have the necessary permissions to connect to the database.

### How do I enable debug logging?
Set the `log_level` configuration to `debug` in your `config.yaml` file or use the environment variable:
```bash
export GOHEAD_LOG_LEVEL=debug
```

### The server isn’t starting. What should I check?
- Ensure that all prerequisites are installed.
- Verify your `config.yaml` file is correctly formatted.
- Check the logs for any specific error messages.

## Development

### Can I contribute to GoHead?
Absolutely! We welcome contributions from the community. Please check the [Contributing Guide](contributing.md) for details on how to get involved.

### How do I run tests?
Run the following command to execute tests:
```bash
make test
```

### What is the recommended development setup?
We recommend using Docker for a consistent development environment. Alternatively, you can run GoHead locally by following the setup instructions in the [Getting Started](getting-started/overview.md) guide.

## Support

### How can I get help?
- Check our [GitHub Discussions](https://github.com/your-org/gohead/discussions).
- Join our [Slack channel](https://slack.your-org.com).
- Review the documentation for troubleshooting tips.

### Where can I report bugs or request features?
Please create an issue on our [GitHub repository](https://github.com/your-org/gohead/issues).

---
If your question isn’t answered here, feel free to reach out via the support channels listed above.
