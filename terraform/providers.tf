terraform {
  required_version = "~> 1.9.0"
  required_providers {
    datadog = {
      source  = "DataDog/datadog"
      version = "~> 3.49" # Allows only the right-most version component to increment
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

provider "datadog" {
  api_url = "https://datadoghq.eu"
  # Provider will read environment variables
  # DD_API_KEY, DD_API_URL, DD_APP_KEY
}
