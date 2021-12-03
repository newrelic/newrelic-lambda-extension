#!/bin/bash

accountId=$1
trustedAccountId=$2
region=$3

echo "Deploying example in region ${region} for NR account ${accountId} with trustedAccountId ${trustedAccountId}"

sam build --use-container

bucket="newrelic-example-${region}-${accountId}"

aws s3 mb --region "${region}" "s3://${bucket}"

sam package --region "${region}" --s3-bucket "${bucket}" --output-template-file packaged.yaml

sam deploy \
	--region "${region}" \
	--template-file packaged.yaml \
	--stack-name Newrelic-Dt-Example \
	--capabilities CAPABILITY_IAM \
	--parameter-overrides "NRAccountId=${accountId}" "TrustedAccountId=${trustedAccountId}"
