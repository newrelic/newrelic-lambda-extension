#!/usr/bin/env bash
set -Eeuo pipefail

EXTENSION_VERSION="2.3.23"
PRIMARY_REGION="us-west-1"
REGIONS=(
  us-west-1
)

hash_file() {
  if command -v md5sum &>/dev/null; then
    md5sum "$1" | awk '{ print $1 }'
  else
    md5 -q "$1"
  fi
}

layer_name_str() {
  local arch=$1
  local arch_part=""
  if [[ "$arch" == "arm64" ]]; then
    arch_part="ARM64"
  fi
  echo "NewRelicLambdaExtension${arch_part}"
}

publish_layer() {
  local layer_archive=$1
  local region=$2
  local arch=$3

  local layer_name
  layer_name=$(layer_name_str "$arch")
  local hash
  hash=$(hash_file "$layer_archive")
  local bucket_name="nr-extension-test-layers-${region}"
  local s3_key="nr-extension/${hash}.${arch}.zip"
  local compat_list=("provided" "provided.al2" "provided.al2023")
  local description="New Relic Layer for provided (${arch}) with New Relic Extension v${EXTENSION_VERSION}"

  echo "Uploading ${layer_archive} to s3://${bucket_name}/${s3_key}" >&2
  aws --region "$region" s3 cp "$layer_archive" "s3://${bucket_name}/${s3_key}" --no-progress >/dev/null
  if [ $? -ne 0 ]; then
    echo "ERROR: AWS S3 upload failed for ${layer_archive} in region ${region}." >&2
    return 1
  fi

  echo "Publishing layer to ${region}" >&2
  local layer_arn
  layer_arn=$(
    aws lambda publish-layer-version \
      --layer-name "${layer_name}" \
      --content "S3Bucket=${bucket_name},S3Key=${s3_key}" \
      --description "${description}" \
      --license-info "Apache-2.0" \
      --compatible-architectures "$arch" \
      --compatible-runtimes "${compat_list[@]}" \
      --region "$region" \
      --output text \
      --query LayerVersionArn
  )

  if [ $? -ne 0 ] || [ -z "$layer_arn" ]; then
    echo "ERROR: Failed to publish Lambda layer for ${arch} in ${region}." >&2
    return 1
  fi

  local layer_version
  layer_version=$(echo "$layer_arn" | awk -F':' '{print $NF}')
  echo "Published version ${layer_version} to ${region}" >&2

  aws lambda add-layer-version-permission \
    --layer-name "${layer_name}" \
    --version-number "$layer_version" \
    --statement-id public \
    --action lambda:GetLayerVersion \
    --principal "*" \
    --region "$region" >/dev/null

  echo "$layer_arn"
}

build_and_publish_for_arch() {
  local arch=$1
  local zip_file="/tmp/newrelic-lambda-extension.${arch}.zip"
  local make_target="zip-${arch}"
  local primary_arn=""

  echo "Processing architecture: ${arch}" >&2
  
  echo "Building layer zip file with 'make ${make_target}'..." >&2
  make --silent "${make_target}"

  echo "Publishing ${arch} layer to all regions from ${zip_file}..." >&2
  for region in "${REGIONS[@]}"; do
    local arn
    arn=$(publish_layer "$zip_file" "$region" "$arch")
    if [[ "$region" == "$PRIMARY_REGION" ]]; then
      primary_arn=$arn
    fi
  done

  echo "$primary_arn"
}

echo "Starting layer publication process..." >&2

x86_arn=$(build_and_publish_for_arch "x86_64")
if [ -z "$x86_arn" ]; then
  echo "ERROR: Could not get ARN for primary region ${PRIMARY_REGION} for x86." >&2
  exit 1
fi
echo "x86_arn=${x86_arn}" >>"$GITHUB_OUTPUT"
echo "Successfully set output for x86_arn" >&2

arm_arn=$(build_and_publish_for_arch "arm64")
if [ -z "$arm_arn" ]; then
  echo "ERROR: Could not get ARN for primary region ${PRIMARY_REGION} for arm64." >&2
  exit 1
fi
echo "arm_arn=${arm_arn}" >>"$GITHUB_OUTPUT"
echo "Successfully set output for arm_arn" >&2

echo "All layers published successfully." >&2
