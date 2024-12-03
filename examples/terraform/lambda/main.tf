terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.78.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

data "aws_caller_identity" "current" {}

data "aws_iam_policy" "newrelic_license_key_policy" {
  arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:policy/ViewNewRelicLicenseKey"
}

resource "aws_iam_role" "newrelic_terraform_example_role" {
  name               = "${var.lambda_function_name}_example_role"
  assume_role_policy = file("./lambda-assume-role-policy.json")
}

resource "aws_iam_role_policy" "newrelic_terraform_example_role_policy" {
  name   = "${var.lambda_function_name}_example_role_policy"
  role   = aws_iam_role.newrelic_terraform_example_role.id
  policy = file("./lambda-policy.json")
}

resource "aws_iam_role_policy_attachment" "newrelic_license_key_policy_attachment" {
  role       = aws_iam_role.newrelic_terraform_example_role.name
  policy_arn = data.aws_iam_policy.newrelic_license_key_policy.arn
}

resource "aws_lambda_function" "newrelic_terraform_example_function" {
  description = "A simple Lambda function, with New Relic telemetry"
  depends_on = [
    aws_cloudwatch_log_group.newrelic_terraform_example_log_group,
    aws_iam_role.newrelic_terraform_example_role,
    aws_iam_role_policy_attachment.newrelic_license_key_policy_attachment
  ]
  filename      = var.lambda_zip_filename
  function_name = var.lambda_function_name
  # The handler for your function needs to be the one provided by the instrumentation layer, below.
  handler = var.wrapper_handler
  role    = aws_iam_role.newrelic_terraform_example_role.arn
  runtime = var.lambda_runtime
  environment {
    variables = {
      # For the instrumentation handler to invoke your real handler, we need this value
      NEW_RELIC_LAMBDA_HANDLER = var.lambda_function_handler
      NEW_RELIC_ACCOUNT_ID     = var.newrelic_account_id
      # Enable NR Lambda extension if the telemetry data are ingested via lambda extension
      NEW_RELIC_LAMBDA_EXTENSION_ENABLED = true
      # Enable Distributed tracing for in-depth monitoring of transactions in lambda (Optional)
      NEW_RELIC_DISTRIBUTED_TRACING_ENABLED = true
    }
  }
  # This layer includes the New Relic Lambda Extension, a sidecar process that sends telemetry,
  # as well as the New Relic Agent, and a handler wrapper that makes integration easy.
  layers = [var.newrelic_layer]
}

resource "aws_cloudwatch_log_group" "newrelic_terraform_example_log_group" {
  name = "/aws/lambda/${var.lambda_function_name}"
  # Lambda functions will auto-create their log group on first execution, but it retains logs forever, which can get expensive.
  retention_in_days = 7
}
