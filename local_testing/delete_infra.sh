#!/bin/bash

# Variables
REGION="us-west-2" # Preferred AWS region

runtimes=("python3.11" "python3.12")
architectures=("x86_64" "arm64")

for arch in "${architectures[@]}"; do
    for runtime in "${runtimes[@]}"; do
        FUNCTION_NAME_SUFFIX="${arch}_${runtime//./}" 
        FUNCTION_NAME="NR_EXTENSION_TEST_LAMBDA_${FUNCTION_NAME_SUFFIX}"
        if aws lambda delete-function --function-name "$FUNCTION_NAME" --region "$REGION"; then
            echo "Successfully deleted $FUNCTION_NAME"
        else
            echo "Failed to delete $FUNCTION_NAME"
        fi
    done
done
