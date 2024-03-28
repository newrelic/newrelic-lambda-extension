#!/bin/bash

accountId=$1
export TF_VAR_newrelic_account_id=$accountId

region=$2
export TF_VAR_aws_region=$region
echo "region set to ${region}"

rm -f function.zip
zip -rq function.zip app.rb

terraform validate .
terraform apply
