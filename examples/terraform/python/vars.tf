variable "aws_region" {
  default = "us-east-1"
}

variable "lambda_function_handler" {
  default = "app.lambda_handler"
}

variable "lambda_function_name" {
  default = "newrelic-terraform-example-python"
}

variable "lambda_runtime" {
  default = "python3.8"
}

variable "lambda_zip_filename" {
  default = "function.zip"
}

variable "newrelic_account_id" {}

variable "newrelic_python_layer" {
  default = "arn:aws:lambda:us-east-1:451483290750:layer:NewRelicPython38:31"
}