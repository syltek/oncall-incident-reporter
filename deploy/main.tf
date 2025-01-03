locals {
  alert_destination = {
    critical = "@slack-platform-datadog-test"
    warning  = "@slack-platform-datadog-test"
  }
}

module "lambda" {
  source = "../../terraform-modules/terraform-aws-lambda"

  metadata = {
    service     = "incident-reporter-test"
    environment = "dev"
    account     = "syltek"

    lambda_config = {
      handler       = "bootstrap"
      memory_size   = 128
      timeout       = 3 # minutes
      filename_path = "lambda_handler.zip"
      environment_variables = {
        SLACK_TOKEN       = var.slack_token
        SLACK_CHANNEL_ID  = var.slack_channel_id
        DD_CLIENT_API_KEY = var.dd_api_key
        DD_CLIENT_APP_KEY = var.dd_app_key
        DEBUG             = "true"
      }
      api = {
        endpoints = [
          {
            path   = "/incident"
            method = "POST"
          },
          {
            path   = "/incident/submit"
            method = "POST"
          }
        ]
      }
    }
  }

  monitoring = {
    lambda_errors = {
      critical = 1
      warning  = 0
    }
    alert_destination = local.alert_destination
  }
}

output "apigw_invoke_url" {
  value = module.lambda.apigateway_invoke_url
}
