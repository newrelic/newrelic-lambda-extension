param (
    [Parameter(Mandatory=$true)]
    [string]$accountId,

    [Parameter(Mandatory=$true)]
    [string]$region
)

Write-Host "region set to $region"

# Call SAM build
& sam build

$bucket = "newrelic-example-$region-$accountId"

# Create S3 bucket
& aws s3 mb --region $region "s3://$bucket"

# Package SAM application
& sam package --region $region --s3-bucket $bucket --output-template-file packaged.yaml

# Deploy CloudFormation stack
& aws cloudformation deploy `
    --region $region `
    --template-file packaged.yaml `
    --stack-name NewrelicExampleDotnet `
    --capabilities CAPABILITY_IAM `
    --parameter-overrides "NRAccountId=$accountId"