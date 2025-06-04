#!/usr/bin/env bash

set -Eeuo pipefail

# https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html
#  Python 3.12 | Amazon Linux 2023
#  Python 3.11 | Amazon Linux 2

BUILD_DIR=python
DIST_DIR=dist

BUCKET_PREFIX=nr-extension-test-layers

EXTENSION_DIST_ZIP_ARM64=$DIST_DIR/extension.arm64.zip
EXTENSION_DIST_ZIP_X86_64=$DIST_DIR/extension.x86_64.zip

PY311_DIST_ARM64=$DIST_DIR/python311.arm64.zip
PY311_DIST_X86_64=$DIST_DIR/python311.x86_64.zip

PY312_DIST_ARM64=$DIST_DIR/python312.arm64.zip
PY312_DIST_X86_64=$DIST_DIR/python312.x86_64.zip

REGIONS_X86=(us-west-2)
REGIONS_ARM=(us-west-2)

EXTENSION_DIST_DIR=extensions
EXTENSION_DIST_ZIP=extension.zip

TMP_ENV_FILE_NAME=nr_tmp_env.sh

function fetch_extension {
    arch=$1
    url="https://github.com/newrelic/newrelic-lambda-extension/releases/download/v2.3.11/newrelic-lambda-extension.${arch}.zip"
    rm -rf $EXTENSION_DIST_DIR $EXTENSION_DIST_ZIP
    curl -L $url -o $EXTENSION_DIST_ZIP
}

function download_extension {
    if [ "$NEWRELIC_LOCAL_TESTING" = "true" ]; then
        case "$1" in
            "x86_64")
                echo "Locally building x86_64 extension"
                make -C ../ dist-x86_64
                ;;
            "arm64")
                echo "Locally building arm64 extension"
                make -C ../ dist-arm64
                ;;
            *)
                echo "No matching architecture"
                return 1 
                ;;
        esac
        cp -r ../extensions .
    else
        fetch_extension "$@"
        unzip "$EXTENSION_DIST_ZIP" -d .
        rm -f "$EXTENSION_DIST_ZIP"
    fi
}

function layer_name_str() {
    rt_part="Custom"
    arch_part=""

    case $1 in
    "python3.11")
      rt_part="Python311"
      ;;
    "python3.12")
      rt_part="Python312"
      ;;
    esac

    case $2 in
    "arm64")
      arch_part="ARM64"
      ;;
    "x86_64")
      arch_part="X86"
      ;;
    esac

    echo "NRTestExtension${rt_part}${arch_part}"
}


function hash_file() {
    if command -v md5sum &> /dev/null ; then
        md5sum $1 | awk '{ print $1 }'
    else
        md5 -q $1
    fi
}


function s3_prefix() {
    name="nr-test-extension"

    case $1 in
    "python3.11")
      name="nr-python3.11"
      ;;
    "python3.12")
      name="nr-python3.12"
      ;;
    esac

    echo $name
}

function publish_layer {
    layer_archive=$1
    region=$2
    runtime_name=$3
    arch=$4

    layer_name=$( layer_name_str $runtime_name $arch )

    hash=$( hash_file $layer_archive | awk '{ print $1 }' )

    bucket_name="${BUCKET_PREFIX}-${region}"
    s3_key="$( s3_prefix $runtime_name )/${hash}.${arch}.zip"

    echo "Uploading ${layer_archive} to s3://${bucket_name}/${s3_key}"
    aws --region "$region" s3 cp $layer_archive "s3://${bucket_name}/${s3_key}"

    echo "Publishing ${runtime_name} layer to ${region}"
    layer_output=$(aws lambda publish-layer-version \
      --layer-name ${layer_name} \
      --content "S3Bucket=${bucket_name},S3Key=${s3_key}" \
      --description "New Relic Test Layer for ${runtime_name} (${arch})" \
      --license-info "Apache-2.0" \
      --region "$region" \
      --output json)

    layer_version=$(echo $layer_output | jq -r '.Version')
    layer_arn=$(echo $layer_output | jq -r '.LayerArn')

    echo "Published ${runtime_name} layer version ${layer_version} to ${region}"
    echo "Layer ARN: ${layer_arn}"
    full_layer_arn="${layer_arn}:${layer_version}"

    echo "Published ${runtime_name} layer version ${layer_version} to ${region}"
    echo "Full Layer ARN with version: ${full_layer_arn}"

    arch_upper=$(echo "$arch" | tr '[:lower:]' '[:upper:]')
    runtime_nodots=$(echo "${runtime_name//./}" | tr '[:lower:]' '[:upper:]')

    env_var_name="LAYER_ARN_${arch_upper}_${runtime_nodots}"
    echo $env_var_name
    declare "$env_var_name=$full_layer_arn"

    echo "export $env_var_name='$full_layer_arn'" >> $TMP_ENV_FILE_NAME
}


function build_python_version {
    version=$1
    arch=$2
    dist_dir=$3

    echo "Building New Relic layer for python$version ($arch)"
    rm -rf $BUILD_DIR $dist_dir
    mkdir -p $DIST_DIR
    pip3 install --no-cache-dir -qU newrelic newrelic-lambda -t $BUILD_DIR/lib/python$version/site-packages
    cp newrelic_lambda_wrapper.py $BUILD_DIR/lib/python$version/site-packages/newrelic_lambda_wrapper.py
    find $BUILD_DIR -name '__pycache__' -exec rm -rf {} +
    download_extension $arch
    zip -rq $dist_dir $BUILD_DIR $EXTENSION_DIST_DIR 
    rm -rf $BUILD_DIR $EXTENSION_DIST_DIR
    echo "Build complete: ${dist_dir}"
}

function publish_python_version {
    dist_dir=$1
    arch=$2
    version=$3
    regions=("${@:4}")

    if [ ! -f $dist_dir ]; then
        echo "Package not found: ${dist_dir}"
        exit 1
    fi

    for region in "${regions[@]}"; do
        publish_layer $dist_dir $region python$version $arch
    done
}

function build_extension_version {
    arch=$1
    dist_dir=$2

    echo "Building New Relic Lambda Extension Layer (x86_64)"
    rm -rf $DIST_DIR
    mkdir -p $DIST_DIR
    download_extension $arch
    zip -rq $dist_dir $EXTENSION_DIST_DIR 
    # rm -rf $EXTENSION_DIST_DIR
    echo "Build complete: ${dist_dir}"
}

function publish_extension_version {
    dist_dir=$1
    arch=$2
    regions=("${@:3}")

    if [ ! -f $dist_dir ]; then
        echo "Package not found: ${dist_dir}"
        exit 1
    fi

    for region in "${regions[@]}"; do
        publish_layer $dist_dir $region extension $arch
    done
}


if [ -f "$TMP_ENV_FILE_NAME" ]; then
    echo "Deleting tmp env file"
    rm -r "$TMP_ENV_FILE_NAME"
else
    echo "File $TMP_ENV_FILE_NAME does not exist."
fi


# Build and publish for python3.11 arm64
echo "Building and publishing for Python 3.11 ARM64..."
build_python_version "3.11" "arm64" $PY311_DIST_ARM64
publish_python_version $PY311_DIST_ARM64 "arm64" "3.11" "${REGIONS_ARM[@]}"

# Build and publish for python3.11 x86_64
echo "Building and publishing for Python 3.11 x86_64..."
build_python_version "3.11" "x86_64" $PY311_DIST_X86_64
publish_python_version $PY311_DIST_X86_64 "x86_64" "3.11" "${REGIONS_X86[@]}"

# Build and publish for python3.12 arm64
echo "Building and publishing for Python 3.12 ARM64..."
build_python_version "3.12" "arm64" $PY312_DIST_ARM64
publish_python_version $PY312_DIST_ARM64 "arm64" "3.12" "${REGIONS_ARM[@]}"

# Build and publish for python3.12 x86_64
echo "Building and publishing for Python 3.12 x86_64..."
build_python_version "3.12" "x86_64" $PY312_DIST_X86_64
publish_python_version $PY312_DIST_X86_64 "x86_64" "3.12" "${REGIONS_X86[@]}"

# Build and publish for Extension ARM64
echo "Building and publishing for Extension ARM64..."
build_extension_version "arm64" $EXTENSION_DIST_ZIP_ARM64
publish_extension_version $EXTENSION_DIST_ZIP_ARM64 "arm64" "${REGIONS_ARM[@]}"

# Build and publish for Extension x86_64
echo "Building and publishing for Extension x86_64..."
build_extension_version "x86_64" $EXTENSION_DIST_ZIP_X86_64
publish_extension_version $EXTENSION_DIST_ZIP_X86_64 "x86_64" "${REGIONS_X86[@]}"