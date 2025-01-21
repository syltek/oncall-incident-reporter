resource "datadog_monitor" "lambda_errors" {
  name  = "[${local.base_name}] AWS Lambda Errors Alert"
  type  = "metric alert"
  query = <<-EOQ
    sum(last_5m):(
      min:aws.lambda.errors{functionname:${local.base_name}, env:${var.environment}}.as_count()
    ) >= 1
  EOQ

  message = templatefile("${path.module}/templates/lambda_errors.md.tftpl", {
    lambda_name       = local.base_name
    environment       = var.environment
    alert_destination = var.alert_destination
    cloudwatch_url    = local.cloudwatch_url
  })

  monitor_thresholds {
    critical = 1
  }

  notify_no_data      = false
  renotify_interval   = 0
  evaluation_delay    = 600
  require_full_window = true

  tags = local.dd_monitor_tags
}
