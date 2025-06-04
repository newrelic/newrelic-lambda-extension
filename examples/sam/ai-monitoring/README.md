# Lambda response streaming: New Relic AI Monitoring

This example was generated using sam init and modified to add newrelic instrumentation with New Relic AI Monitoring capabilities. We've used modelId: 'anthropic.claude-v2', but any AI models can be utilized similarly. For more information on AI monitoring with New Relic, refer to the documentation[https://docs.newrelic.com/install/ai-monitoring/?agent-lang=nodejs].

Important: this application uses various AWS services and there are costs associated with these services after the Free Tier usage - please see the [AWS Pricing page](https://aws.amazon.com/pricing/) for details. You are responsible for any AWS costs incurred. No warranty is implied in this example.

## Requirements

* [Create an AWS account](https://portal.aws.amazon.com/gp/aws/developer/registration/index.html) if you do not already have one and log in. The IAM user that you use must have sufficient permissions to make necessary AWS service calls and manage AWS resources.
* [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html) installed and configured
* [Git Installed](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)
* [AWS Serverless Application Model](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html) (AWS SAM) installed

##  Initial Setup and Deployment Instructions Using AWS SAM
### 1. Clone the repository

```bash
git clone https://github.com/newrelic/newrelic-lambda-extension.git
```

Navigate to the example pattern directory:

```bash
cd examples/sam/ai-monitoring/
```

### 2. Build the AWS SAM Application

Before deploying, you need to build the application using:

```bash
sam build
```

This command compiles your application into the `.aws-sam/build` directory, preparing it for deployment by resolving dependencies and building necessary resources.

### 3. Deploy the AWS Resources
Deploy the AWS resources using AWS SAM:

```bash
sam deploy --guided
```

During this phase, you'll answer several prompts:

- **Stack name**: Choose a name for your stack.
- **AWS Region**: Select the AWS region (default region recommended).
- **IAM roles**: Permit SAM CLI to create IAM roles for your application resources.

After completing the guided deployment once, a `samconfig.toml` file will be generated with saved configurations.

### 4. Subsequent Deployments
For future deployments, you can simply use:

```bash
sam deploy
```

This will utilize saved defaults from your `samconfig.toml` file.

## Testing the Deployment

To test the streaming feature, the `curl` command includes `--user` and `--aws-sigv4` because our Lambda function is secured with the `AWS_IAM` authentication type. This requires signing the request with AWS credentials to ensure proper authentication and authorization.

```bash
curl --request GET https://<url>.lambda-url.<Region>.on.aws/ \
--user AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
--aws-sigv4 'aws:amz:<Region>:lambda' \
-d '{"prompt": "hello! how are you?"}'
```
Ensure you replace `<url>` and `<Region>` with your specific deployment details. For more details, refer to [AWS IAM Documentation](https://docs.aws.amazon.com/lambda/latest/dg/urls-auth.html#urls-auth-iam).



## Cleanup Resources

To delete the stack and associated resources, execute:

```bash
sam delete
```

Confirm the deletion by entering `Y` when prompted to ensure clean removal of your deployed stack and resources.
