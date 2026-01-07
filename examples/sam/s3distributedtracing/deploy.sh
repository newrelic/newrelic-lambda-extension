#!/bin/bash

accountId=$1
trustedAccountId=$2
browserAppId=$3
browserAgentId=$4
browserLicenseKey=$5
newRelicLicenseKey=$6
region=$7

echo "Deploying S3 distributed tracing example in region ${region} for NR account ${accountId} with trustedAccountId ${trustedAccountId}"

sam build --use-container

bucket="newrelic-s3-example-${region}-${accountId}"

aws s3 mb --region "${region}" "s3://${bucket}"

sam package --region "${region}" --s3-bucket "${bucket}" --output-template-file packaged.yaml

aws cloudformation deploy \
	--region "${region}" \
	--template-file packaged.yaml \
	--stack-name Newrelic-Dt-S3-Example \
	--capabilities CAPABILITY_IAM \
	--parameter-overrides "NRAccountId=${accountId}" "TrustedAccountId=${trustedAccountId}" "BrowserAppId=${browserAppId}" "BrowserAgentId=${browserAgentId}" "BrowserLicenseKey=${browserLicenseKey}" "NewRelicLicenseKey=${newRelicLicenseKey}"
