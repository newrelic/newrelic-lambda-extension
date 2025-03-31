# Lambda Response streaming: NR AI MONITORING

This example showcases the utility of response streaming and introduces AI monitoring through New Relic (NR). We've used modelId: 'anthropic.claude-v2', but any AI models can be utilized similarly. For more information on AI monitoring with New Relic, refer to the documentation[https://docs.newrelic.com/install/ai-monitoring/?agent-lang=nodejs]. 

Important: this application uses various AWS services and there are costs associated with these services after the Free Tier usage - please see the [AWS Pricing page](https://aws.amazon.com/pricing/) for details. You are responsible for any AWS costs incurred. No warranty is implied in this example.

## Requirements

* [Create an AWS account](https://portal.aws.amazon.com/gp/aws/developer/registration/index.html) if you do not already have one and log in. The IAM user that you use must have sufficient permissions to make necessary AWS service calls and manage AWS resources.
* [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html) installed and configured
* [Git Installed](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)
* [AWS Serverless Application Model](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html) (AWS SAM) installed

## Deployment Instructions

1. Create a new directory, navigate to that directory in a terminal and clone the GitHub repository:

```
git clone https://github.com/newrelic/newrelic-lambda-extension.git
```

2. Change directory to the pattern directory:

    ```
    cd examples/sam/ai-monitoring/
    ```

3. From the command line, use AWS SAM to deploy the AWS resources for the pattern as specified in the template.yml file:

    ```
    sam deploy --guided
    ```

4. During the prompts:
    * Enter a stack name
    * Enter the desired AWS Region - AWS CLI default region is recommended
    * Allow SAM CLI to create IAM roles with the required permissions.

5. Once you have run `sam deploy --guided` mode once and saved arguments to a configuration file `samconfig.toml`, you can use `sam deploy` in future to use these defaults.

## Testing

1.	Use curl with your AWS credentials to view the streaming response as the url uses AWS Identity and Access Management (IAM) for authorization. Replace the URL and Region parameters for your deployment.

    ```
    curl --request GET https://<url>.lambda-url.<Region>.on.aws/ --user AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY --aws-sigv4 'aws:amz:<Region>:lambda' -d '{"prompt": "hello! how are you?"}'
    ```



## Cleanup
 
1. Delete the stack, Enter `Y` to confirm deleting the stack and folder.
    ```
    sam delete
    ```