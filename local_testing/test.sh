#!/bin/bash

# Variables
FUNCTION_FILE="function.zip"
ROLE_NAME="nr_extension_test_lambda_execution_role"
REGION="us-west-2" # Preferred AWS region
TRUST_POLICY="trust-policy.json"
lambda_result_file="lambda_output.txt"

NEW_RELIC_ACCOUNT_ID="${NEW_RELIC_ACCOUNT_ID}"
NEW_RELIC_LAMBDA_EXTENSION_ENABLED="True"
NEW_RELIC_LAMBDA_HANDLER="${NEW_RELIC_LAMBDA_HANDLER}"
NEW_RELIC_LICENSE_KEY_SECRET="${NEW_RELIC_LICENSE_KEY_SECRET}"
NEW_RELIC_LOG_ENDPOINT="${NEW_RELIC_LOG_ENDPOINT}"
NEW_RELIC_TELEMETRY_ENDPOINT="${NEW_RELIC_TELEMETRY_ENDPOINT}"

source nr_tmp_env.sh

zip function.zip function.py

role_exists=$(aws iam get-role --role-name "$ROLE_NAME" --query 'Role.Arn' --output text --region "$REGION" 2>&1)
if [[ $role_exists == arn:aws:iam::* ]]; then
    echo "IAM role already exists."
    EXECUTION_ROLE_ARN=$role_exists
else
    EXECUTION_ROLE_ARN=$(aws iam create-role --role-name "$ROLE_NAME" --assume-role-policy-document "file://$TRUST_POLICY" --query 'Role.Arn' --output text --region "$REGION")
    if [ $? -ne 0 ]; then
        echo "Error creating IAM role."
        exit 1
    fi

    aws iam attach-role-policy --role-name "$ROLE_NAME" --policy-arn "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole" --region "$REGION"
    echo "Attached policy to role."

    echo "Waiting for IAM role to propagate."
    sleep 10
fi

if [ ! -f "$FUNCTION_FILE" ]; then
    echo "Lambda function file $FUNCTION_FILE does not exist."
    exit 1
fi

runtimes=("python3.11" "python3.12")
architectures=("x86_64" "arm64")

for arch in "${architectures[@]}"; do
    for runtime in "${runtimes[@]}"; do
        FUNCTION_NAME_SUFFIX="${arch}_${runtime//./}" 
        FUNCTION_NAME="NR_EXTENSION_TEST_LAMBDA_${FUNCTION_NAME_SUFFIX}"
        arch_upper=$(echo "$arch" | tr '[:lower:]' '[:upper:]')
        runtime_nodots=$(echo "${runtime//./}" | tr '[:lower:]' '[:upper:]')

        env_var_name="LAYER_ARN_${arch_upper}_${runtime_nodots}"
        final_layer_arn="${!env_var_name}"
        echo "$final_layer_arn"
        if aws lambda get-function --function-name "$FUNCTION_NAME" --region "$REGION" 2>/dev/null; then
            echo "Lambda function $FUNCTION_NAME already exists."
        else
            FUNCTION_ARN=$(aws lambda create-function \
                --function-name "$FUNCTION_NAME" \
                --zip-file "fileb://$FUNCTION_FILE" \
                --handler "function.handler" \
                --runtime "$runtime" \
                --architectures "$arch" \
                --role "$EXECUTION_ROLE_ARN" \
                --layers "$final_layer_arn" \
                --environment Variables="{NEW_RELIC_ACCOUNT_ID=$NEW_RELIC_ACCOUNT_ID,NEW_RELIC_LAMBDA_EXTENSION_ENABLED=$NEW_RELIC_LAMBDA_EXTENSION_ENABLED,NEW_RELIC_LAMBDA_HANDLER=$NEW_RELIC_LAMBDA_HANDLER,NEW_RELIC_LICENSE_KEY_SECRET=$NEW_RELIC_LICENSE_KEY_SECRET,NEW_RELIC_LOG_ENDPOINT=$NEW_RELIC_LOG_ENDPOINT,NEW_RELIC_TELEMETRY_ENDPOINT=$NEW_RELIC_TELEMETRY_ENDPOINT}" \
                --query 'FunctionArn' \
                --output text \
                --region "$REGION")
            if [ $? -ne 0 ]; then
                echo "Error creating Lambda function $FUNCTION_NAME."
                exit 1
            fi
            echo "Created lambda function: $FUNCTION_ARN"

        fi
        sleep 10
        aws lambda invoke --function-name "$FUNCTION_NAME" --region "$REGION" --payload '{}' "$lambda_result_file"
        if [ $? -eq 0 ]; then
            echo "Lambda function invoked successfully. Output in $lambda_result_file"
        else
            echo "Error invoking Lambda function."
        fi
    done
done
