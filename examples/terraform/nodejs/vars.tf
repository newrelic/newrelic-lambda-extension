variable "aws_region" {
  default = "us-east-1"
}

variable "lambda_function_handler" {
  default = "app.lambdaHandler"
}

variable "lambda_function_name" {
  default = "newrelic-terraform-example-nodejs"
}

variable "lambda_runtime" {
  default = "nodejs12.x"
}

variable "lambda_zip_filename" {
  default = "function.zip"
}

variable "newrelic_account_id" {}

variable "newrelic_nodejs_layer" {
  default = "arn:aws:lambda:us-east-1:451483290750:layer:NewRelicNodeJS12X:44"
}