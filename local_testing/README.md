## Local Testing

To test the extension on a AWS test account, follow the steps:
1. Configure the credentials for the AWS test account `aws configure`
2. Modify the fetch_extension url in publish.sh to your forked repo's extension release url
3. Run `./publish.sh` to publish the layers to your test account `us-west-2` region
4. Publish script will create 4 lambda layers for runtimes - Python 3.12 [[Amazon Linux 2023](https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html)] & Python 3.11 [[Amazon Linux 2](https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html)] and architectures - x86 & ARM
5. Run `./test.sh` to create lambda with test layer published in step 2
6. Go to your AWS test account and check the logs of the lambda with suffix - `NR_EXTENSION_TEST_LAMBDA_` for any error in extension
