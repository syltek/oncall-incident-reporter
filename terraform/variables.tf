variable "service_name" {
  type        = string
  description = "The name of the service"
}

variable "environment" {
  type        = string
  description = "The environment to deploy the lambda function to"
}

variable "aws_account" {
  type        = string
  description = "The AWS account"
}

variable "aws_region" {
  type        = string
  description = "The region of the AWS account"
}

variable "handler" {
  type        = string
  description = "The handler of the lambda function"
  default     = "bootstrap"
}

variable "memory_size" {
  type        = number
  description = "The memory size of the lambda function"
  default     = 128
}

variable "timeout" {
  type        = number
  description = "The timeout of the lambda function"
  default     = 300
}

variable "filename_path" {
  type        = string
  description = "The path to the lambda function zip file"
  default     = "lambda_handler.zip"
}

variable "environment_variables" {
  type        = map(string)
  description = "The environment variables of the lambda function"
  default     = {}
}

variable "alert_destination" {
  type        = string
  description = "The destination of the alert"
  default     = "@slack-platform-datadog-test"
}

variable "endpoints" {
  type = list(object({
    method = string
    path   = string
  }))
  description = "The endpoints of the API Gateway"

  default = [
    {
      method = "POST"
      path   = "/dev/incident"
    },
    {
      method = "GET"
      path   = "/dev/incident/submit"
    }
  ]
}

variable "datadog_api_url" {
  type        = string
  description = "The API URL of the Datadog API"
  default     = "https://api.datadoghq.eu/"
}

variable "tags" {
  type        = map(string)
  description = "The tags of the lambda function"
  default     = {}
}
