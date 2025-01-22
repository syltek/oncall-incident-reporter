<p align="center">
  <img src="doc/img/playtomic-logo.png" alt="Playtomic Logo" width="400"/>
</p>

<h1 align="center">Oncall Incident Reporter</h1>

<div align="center">

[![Go Report Card](https://goreportcard.com/badge/github.com/syltek/oncall-incident-reporter)](https://goreportcard.com/report/github.com/syltek/oncall-incident-reporter)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/syltek/oncall-incident-reporter)](https://golang.org/dl/)

</div>

## Introduction

A Golang AWS Lambda application that bridges Slack and Datadog's on-call system, allowing teams to trigger incidents directly
from Slack commands. When users invoke a slash command, they're presented with a customizable form to input incident details.
Upon submission, the application creates an error event in Datadog, which can trigger monitors and page on-call teams.

## Features

- 🚀 Slack slash command integration for incident reporting
- 📝 Interactive modal forms for incident details (configurable via YAML)
- 🔄 Automatic Datadog error event creation
- 🔔 Integration with Datadog's on-call system
- 🔒 Slack signature validation following the [Slack API documentation](https://api.slack.com/authentication/verifying-requests-from-slack#validating-a-request)
- 📊 Structured logging with sensitive data redaction
- ☁️ Ready for AWS Lambda deployment
- 📚 Documentation and examples for configuration

## How It Works

1. User triggers a slash command in Slack
2. Application presents a customizable modal form
3. User fills incident details
4. Application creates an error event in Datadog
5. Datadog monitor (example provided) detects the event and pages on-call team

## Prerequisites

- Go `1.23` or higher
- [Task](https://taskfile.dev/) - Task runner/build tool
- Slack Workspace with admin access
- Datadog account with API and Application keys
- AWS account with Lambda and API Gateway access

## Installation

1. Clone the repository:

```bash
git clone https://github.com/syltek/oncall-incident-reporter.git
cd oncall-incident-reporter
```

2. Install task

```bash
brew install task
```

3. Copy config example configuration and customize it

```bash
cp config.example.yaml config.yaml
```

### Environment Variables

- `SLACK_TOKEN` - Slack Bot User OAuth Token
- `CONFIG_FILE` - Path to configuration file (default: config.yaml)
- `DEBUG` - Enable debug logging (true/false)
- `LOCAL` - Run in local development mode (true/false)

## Deployment

### AWS Lambda Setup

The application is designed to work with AWS Lambda and API Gateway. The handler uses `events.APIGatewayProxyRequest` and
`events.APIGatewayProxyResponse` for seamless integration.

Example Terraform configurations are provided in the [terraform](./terraform) directory, including:
- Lambda function configuration
- API Gateway setup
- Required IAM roles and policies
- Datadog monitor example

### Deploy with Terraform

See [terraform_deployment.md](doc/terraform_deployment.md)

## Usage

### Local Development

Start the application locally with hot reload:

```bash
task start
```

## Architecture

The application is structured into several key packages:

- `cmd/` - Application entry point and main configuration
- `internal/` - Internal application code
  - `clients/` - Creates the external clients (Slack, Datadog).
  - `config/` - Configuration management
  - `handlers/` - Request handlers
  - `middleware/` - HTTP middleware
  - `router/` - HTTP routing
  - `service/` - Application services. Contains the external clients.
  - `slackmodal/` - Slack modal handling
- `pkg/` - Shared packages
  - `errors/` - Error handling
  - `logutil/` - Logging utilities

## Slack Command Setup

For detailed instructions on setting up the Slack command integration, including:
- Creating a Slack App
- Configuring slash commands
- Setting up interactivity endpoints
- Configuring authentication
- Testing the integration

See [Slack Command Setup Guide](doc/example_slack_command.md)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## About Playtomic

<img align="right" src="doc/img/playtomic-logo.png" alt="Playtomic Logo" width="100"/>

This project is sponsored and maintained by [Playtomic](https://playtomic.io), the leading sports facility booking platform
in the world. We're passionate about technology and sports, building tools that help millions of players enjoy their favorite
raquet sports.

- Website: https://playtomic.io
- Tech Blog: https://tech.playtomic.io
- Careers: https://jobs.playtomic.io

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE.md) file for details.

## Acknowledgments

- [Slack Go SDK](https://github.com/slack-go/slack)
- [Datadog Go Client](https://github.com/DataDog/datadog-api-client-go)
- [Gorilla Mux](https://github.com/gorilla/mux)
- [Viper](https://github.com/spf13/viper)
- [Zap](https://github.com/uber-go/zap)

## Support

For support, please open an issue in the GitHub issue tracker.
