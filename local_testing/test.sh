#!/bin/bash

# Variables
LAYER_NAME="MyLambdaLayer"
LAYER_ZIP="newrelic_lambda_layer.zip"
FUNCTION_NAME="Saket_sample_Integration_test"
FUNCTION_FILE="function.zip"
ROLE_NAME="LambdaExecutionRole"
POLICY_NAME="AWSLambdaBasicExecutionRole"
REGION="us-east-1" # Change to your preferred AWS region
LOG_GROUP_NAME="/aws/lambda/${FUNCTION_NAME}"
TRUST_POLICY="trust-policy.json"

# Create trust policy file
cat > $TRUST_POLICY << EOL
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOL

# Create IAM Role for Lambda Execution
EXECUTION_ROLE_ARN=$(aws iam create-role --role-name $ROLE_NAME --assume-role-policy-document file://$TRUST_POLICY --query 'Role.Arn' --output text)

# Attach the basic execution policy to the role
aws iam attach-role-policy --role-name $ROLE_NAME --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

# Wait a bit for the IAM role to propagate
sleep 10

# Create Lambda Layer
LAYER_ARN=$(aws lambda publish-layer-version --layer-name $LAYER_NAME --region $REGION --zip-file fileb://$LAYER_ZIP --query 'LayerVersionArn' --output text)
echo $LAYER_ARN
echo "Created lambda layer"
# Create Lambda Function
FUNCTION_ARN=$(aws lambda create-function --function-name $FUNCTION_NAME --zip-file fileb://$FUNCTION_FILE --handler function.handler --runtime python3.8 --role $EXECUTION_ROLE_ARN --layers $LAYER_ARN --query 'FunctionArn' --output text --region $REGION)
echo $FUNCTION_ARN
echo "Created lambda function"
# Invoke Lambda Function
aws lambda invoke --function-name $FUNCTION_NAME --region $REGION --payload '{}' output.txt 
echo "Lambda Invoke function successful"

Wait a bit for logs to generate
sleep 10

# Get Logs
LOG_STREAM_NAME=$(aws logs describe-log-streams --log-group-name "$LOG_GROUP_NAME" --region "$REGION" --max-items 1 --order-by LastEventTime --descending --query 'logStreams[0].logStreamName' --output text)
echo $LOG_STREAM_NAME
# Assuming LOG_STREAM_NAME might have the value "None" at the end or be invalid
CLEANED_LOG_STREAM_NAME=$(echo "$LOG_STREAM_NAME" | sed 's/[[:space:]]*None$//' | tr -d '\r')

if [[ -z "$CLEANED_LOG_STREAM_NAME" ]]; then
  echo "Log stream name is invalid or not found."
else
    echo "cleaned log"
    echo $CLEANED_LOG_STREAM_NAME
    aws logs get-log-events --log-group-name "$LOG_GROUP_NAME" --region "$REGION" --log-stream-name "$CLEANED_LOG_STREAM_NAME" --output text
fi


# Cleanup steps (Lambda function, layer, IAM role)
aws lambda delete-function --function-name $FUNCTION_NAME --region $REGION
LAYER_VERSION=$(echo $LAYER_ARN | awk -F':' '{print $NF}')
aws lambda delete-layer-version --layer-name $LAYER_NAME --version-number $LAYER_VERSION --region $REGION
aws iam detach-role-policy --role-name $ROLE_NAME --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
aws iam delete-role --role-name $ROLE_NAME

# Optional: Delete Log Group
aws logs delete-log-group --log-group-name $LOG_GROUP_NAME

echo "Lambda function, layer, and IAM role cleanup completed."
