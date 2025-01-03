# Common Lambda resources
resource "aws_lambda_function" "lambda" {
  function_name = var.service_name
  role          = aws_iam_role.lambda_permissions.arn
  handler       = var.handler
  runtime       = "provided.al2"

  memory_size = var.memory_size
  timeout     = var.timeout

  filename         = var.filename_path
  source_code_hash = filebase64sha256(var.filename_path)

  # Add environment variables configuration
  environment {
    variables = var.environment_variables
  }

  tags = local.common_tags
}

# CloudWatch Logs
resource "aws_cloudwatch_log_group" "lambda_logs" {
  name              = "/aws/lambda/${local.base_name}"
  retention_in_days = 30
  tags              = local.common_tags
}

resource "aws_iam_role" "lambda_permissions" {
  name               = local.base_name
  assume_role_policy = data.aws_iam_policy_document.assume_role_policy.json
  tags               = local.common_tags
}

resource "aws_iam_role_policy" "lambda_access_logs" {
  name   = "${local.base_name}-allow-access-logs"
  role   = aws_iam_role.lambda_permissions.id
  policy = data.aws_iam_policy_document.cloudwatch_logs.json
}

data "aws_iam_policy_document" "assume_role_policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com", "apigateway.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "cloudwatch_logs" {
  # Allow Cloudwatch logging
  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:DescribeLogGroups",
      "logs:DescribeLogStreams",
      "logs:PutLogEvents",
      "logs:GetLogEvents",
      "logs:FilterLogEvents",
    ]
    # We can't scope down to the specific log group because is created in this same code
    # Due to a terraform limitation.
    resources = ["arn:aws:logs:*:*:*"]
  }
}
