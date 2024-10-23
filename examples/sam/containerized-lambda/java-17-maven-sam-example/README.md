# java-17-maven-sam-example

This project contains source code and supporting files for a serverless application that you can deploy with the SAM CLI. It includes:

- `HelloWorldFunction/src/main`: Code for the application's Lambda function and Project Dockerfile.
- `events`: Invocation events that you can use to invoke the function.
- `HelloWorldFunction/src/test`: Unit tests for the application code.
- `template.yaml`: A template that defines the application's AWS resources.

## Getting Started

### Prerequisites

1. **Install SAM CLI**: [Installation instructions](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
2. **Install Docker**: [Docker Community Edition installation guide](https://hub.docker.com/search/?type=edition&offering=community)
3. **Install Java 17**: [Java 17 installation](https://docs.aws.amazon.com/corretto/latest/corretto-17-ug/downloads-list.html)
4. **Install Maven**: [Maven installation](https://maven.apache.org/install.html)

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
   curl http://localhost:3000/
   ```

## Fetch Logs

1. **Fetch Lambda Logs**:
   ```bash
   sam logs -n HelloWorldFunction --stack-name java-17-maven-sam-example --tail
   ```

## Run Unit Tests

1. **Run Tests**:
   ```bash
   cd HelloWorldFunction
   mvn test
   ```

## Cleanup

1. **Delete Deployed Application**:
   ```bash
   sam delete --stack-name java-17-maven-sam-example
   ```

## Resources

- [AWS SAM Developer Guide](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/what-is-sam.html)
- [AWS Serverless Application Repository](https://aws.amazon.com/serverless/serverlessrepo/)