# python-sam-example

This project contains source code for a "Hello World" serverless application that you can deploy with the SAM CLI. It includes:

- `hello_world`: Code for the application's Lambda function and Dockerfile.
- `tests`: Unit tests for the application code.
- `template.yaml`: Defines the application's AWS resources.

## Getting Started

### Prerequisites

1. **Install SAM CLI**: [Installation instructions](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
2. **Install Docker**: [Docker Community Edition installation guide](https://hub.docker.com/search/?type=edition&offering=community)
3. **Install Python 3**: [Python 3 installation](https://www.python.org/downloads/)

### Option 1: Create a New SAM Project

1. **Initialize Project**:
   ```bash
   sam init --package-type Image
   ```
2. **Select Template**: Choose `AWS Quick Start Templates` and `Hello World Example`.
3. **Configure Project**: Follow the prompts.

### Option 2: Clone Existing Repository

1. **Clone Repository**:
   ```bash
   git clone https://github.com/newrelic/newrelic-lambda-extension.git
   ```

## Deploy the Application

1. **Build Application**:
   ```bash
   sam build
   ```
2. **Deploy Application**:
   ```bash
   sam deploy --guided
   ```

## Local Development

1. **Build Locally**:
   ```bash
   sam build
   ```
2. **Invoke Function Locally**:
   ```bash
   sam local invoke HelloWorldFunction
   ```
3. **Start Local API**:
   ```bash
   sam local start-api
   curl http://localhost:3000/hello
   ```

## Fetch Logs

1. **Fetch Lambda Logs**:
   ```bash
   sam logs -n HelloWorldFunction --stack-name "python-sam-example" --tail
   ```

## Run Unit Tests

1. **Install Test Dependencies**:
   ```bash
   pip install pytest pytest-mock --user
   ```
2. **Run Tests**:
   ```bash
   python -m pytest tests/ -v
   ```

## Cleanup

1. **Delete Deployed Application**:
   ```bash
   sam delete --stack-name "python-sam-example"
   ```

## Resources

- [AWS SAM Developer Guide](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/what-is-sam.html)
- [AWS Serverless Application Repository](https://aws.amazon.com/serverless/serverlessrepo/)
