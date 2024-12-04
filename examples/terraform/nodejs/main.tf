module "nodejs_test_function" {
  source = "../lambda"
  aws_region = var.aws_region
  lambda_function_handler = "app.lambdaHandler"
  wrapper_handler = "newrelic-lambda-wrapper.handler"
  lambda_function_name = "newrelic-terraform-example-nodejs"
  lambda_runtime = "nodejs20.x"
  lambda_zip_filename = "function.zip"
  newrelic_account_id = var.newrelic_account_id
  newrelic_layer = "arn:aws:lambda:${var.aws_region}:451483290750:layer:NewRelicNodeJS20X:44"
}