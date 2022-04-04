module "python_test_function" {
  source = "../lambda"
  aws_region = var.aws_region
  lambda_function_handler = "app.lambda_handler"
  wrapper_handler = "newrelic_lambda_wrapper.handler"
  lambda_function_name = "newrelic-terraform-example-python"
  lambda_runtime = "python3.8"
  lambda_zip_filename = "function.zip"
  newrelic_account_id = var.newrelic_account_id
  newrelic_layer = "arn:aws:lambda:${var.aws_region}:451483290750:layer:NewRelicPython38:39"
}
