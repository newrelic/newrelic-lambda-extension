# python-sam-example

This project contains source code for a "Hello World" serverless application that you can deploy with the SAM CLI. It includes the following files and folders:

- hello_world - Code for the application's Lambda function and Project Dockerfile.
- tests - Unit tests for the application code.
- template.yaml - A template that defines the application's AWS resources.

The application uses several AWS resources, including Lambda functions and an API Gateway API. These resources are defined in the `template.yaml` file in this project.


## Getting started

There are two ways to start using this serverless application example:

### Option 1: Create a new SAM project

You can create your own SAM project using the `sam init` command:

1. **Set up your environment**: Before you start, make sure that you have the SAM CLI and Docker installed on your machine.
   - Install the SAM CLI following these [installation instructions](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html).
   - Install Docker using the [Docker Community Edition installation guide](https://hub.docker.com/search/?type=edition&offering=community).

2. **Initiate a new project**:
   Run the following command to create a new serverless application with a Docker image package type:

   ```bash
   sam init --package-type Image
   ```

3. **Select the desired template**: Choose `AWS Quick Start Templates` and then select the `Hello World Example`.

4. **Configure your project**: Follow the prompts to set up your project, adding any additional features if necessary.

### Option 2: Clone an existing repository

Alternatively, you can clone this existing repository and deploy the application directly.

After cloning the repository or creating a new SAM project, you can then build and deploy your application as described in the following sections.


The application uses several AWS resources, including Lambda functions and an API Gateway API. These resources are defined in the `template.yaml` file in this project. You can update the template to add AWS resources through the same deployment process that updates your application code.


## Deploy the sample application


The Serverless Application Model Command Line Interface (SAM CLI) is an extension of the AWS CLI that adds functionality for building and testing Lambda applications. It uses Docker to run your functions in an Amazon Linux environment that matches Lambda. It can also emulate your application's build environment and API.

To use the SAM CLI, you need the following tools.

* SAM CLI - [Install the SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
* Docker - [Install Docker community edition](https://hub.docker.com/search/?type=edition&offering=community)

You may need the following for local testing.
* [Python 3 installed](https://www.python.org/downloads/)

To build and deploy your application for the first time, run the following in your shell:

```bash
sam build
sam deploy --guided
```

The `sam build` command builds the application, and the `sam deploy --guided` command prompts you to configure the deployment settings. The guided deployment will also save your settings to `samconfig.toml` for future deployments.

The first command will build a docker image from a Dockerfile and then copy the source of your application inside the Docker image. The second command will package and deploy your application to AWS, with a series of prompts:

* **Stack Name**: The name of the stack to deploy to CloudFormation. This should be unique to your account and region, and a good starting point would be something matching your project name.
* **AWS Region**: The AWS region you want to deploy your app to.
* **Confirm changes before deploy**: If set to yes, any change sets will be shown to you before execution for manual review. If set to no, the AWS SAM CLI will automatically deploy application changes.
* **Allow SAM CLI IAM role creation**: Many AWS SAM templates, including this example, create AWS IAM roles required for the AWS Lambda function(s) included to access AWS services. By default, these are scoped down to minimum required permissions. To deploy an AWS CloudFormation stack which creates or modifies IAM roles, the `CAPABILITY_IAM` value for `capabilities` must be provided. If permission isn't provided through this prompt, to deploy this example you must explicitly pass `--capabilities CAPABILITY_IAM` to the `sam deploy` command.
* **Save arguments to samconfig.toml**: If set to yes, your choices will be saved to a configuration file inside the project, so that in the future you can just re-run `sam deploy` without parameters to deploy changes to your application.

You can find your API Gateway Endpoint URL in the output values displayed after deployment.

## Use the SAM CLI to build and test locally

Build your application with the `sam build` command.

```bash
python-sam-example$ sam build
```

The SAM CLI builds a docker image from a Dockerfile and then installs dependencies defined in `hello_world/requirements.txt` inside the docker image. The processed template file is saved in the `.aws-sam/build` folder.

You can invoke the function locally with the `sam local invoke` command:

```bash
python-sam-example$ sam local invoke HelloWorldFunction
```

The SAM CLI can also emulate your application's API. Run the API locally with the following command:

```bash
python-sam-example$ sam local start-api
python-sam-example$ curl http://localhost:3000/hello
```

## Add a resource to your application
The application template uses AWS Serverless Application Model (AWS SAM) to define application resources. AWS SAM is an extension of AWS CloudFormation with a simpler syntax for configuring common serverless application resources such as functions, triggers, and APIs. For resources not included in [the SAM specification](https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md), you can use standard [AWS CloudFormation](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-template-resource-type-ref.html) resource types.

## Fetch, tail, and filter Lambda function logs

To simplify troubleshooting, SAM CLI has a command called `sam logs`. `sam logs` lets you fetch logs generated by your deployed Lambda function from the command line. In addition to printing the logs on the terminal, this command has several nifty features to help you quickly find the bug.

`NOTE`: This command works for all AWS Lambda functions; not just the ones you deploy using SAM.

```bash
python-sam-example$ sam logs -n HelloWorldFunction --stack-name "python-sam-example" --tail
```

You can find more information and examples about filtering Lambda function logs in the [SAM CLI Documentation](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-logging.html).


## Unit tests

Run unit tests using pytest:

```bash
python-sam-example$ pip install pytest pytest-mock --user
python-sam-example$ python -m pytest tests/ -v
```

## Cleanup

To delete the deployed "Hello World" application, run:

```bash
sam delete --stack-name "python-sam-example"
```

## Resources

For more information about developing with SAM, visit the [AWS SAM developer guide](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/what-is-sam.html).

Check out the [AWS Serverless Application Repository](https://aws.amazon.com/serverless/serverlessrepo/) to find ready-to-use serverless applications beyond simple samples.