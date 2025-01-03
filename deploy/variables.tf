variable "aws_region" {
  type        = string
  default     = "eu-central-1"
  description = "The AWS region to deploy the lambda function to"
}

variable "dd_api_key" {
  type        = string
  description = "The Datadog API key"
}

variable "dd_app_key" {
  type        = string
  description = "The Datadog app key"
}

variable "slack_token" {
  type        = string
  description = "The Slack token"
  sensitive   = true
}

variable "slack_channel_id" {
  type        = string
  description = "The Slack channel ID"
}
