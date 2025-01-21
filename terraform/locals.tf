locals {
  base_name = "${var.environment}-${var.service_name}"
  dd_monitor_tags = [
    for key, value in local.common_tags : "${key}:${value}"
  ]

  monitor_base_url = "https://app.datadoghq.eu/monitors"
  cloudwatch_url   = "https://${var.aws_region}.console.aws.amazon.com/cloudwatch/home?region=${var.aws_region}#logsV2:log-groups/log-group/%2Faws%2Flambda%2F${var.environment}-${var.service_name}"
  common_tags = merge({
    "managed_by" = "terraform",
    "service"    = var.service_name
    "account"    = var.aws_account
    "env"        = var.environment
  }, var.tags != null ? var.tags : {})
}
