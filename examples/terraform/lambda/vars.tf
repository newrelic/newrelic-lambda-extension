variable "aws_region" {
  type = string
}

variable "lambda_function_handler" {
  type = string
}

variable "lambda_function_name" {
  type = string
}

variable "wrapper_handler" {
  type = string
}

variable "lambda_runtime" {
  type = string
}

variable "lambda_zip_filename" {
  type = string
}

variable "newrelic_account_id" {
  type = string
}

variable "newrelic_layer" {
  type = string
}
