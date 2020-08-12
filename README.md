[![Community Project header](https://github.com/newrelic/opensource-website/raw/master/src/images/categories/Community_Project.png)](https://opensource.newrelic.com/oss-category/#community-project)

# newrelic-lambda-extension [![Build Status][1]][2]

[1]: https://circleci.com/gh/newrelic/newrelic-lambda-extension.svg?style=svg
[2]: https://circleci.com/gh/newrelic/newrelic-lambda-extension

`newrelic-lambda-extension` is TODO: write me


## Installation

To install the extension, simply include the layer with your instrumented Lambda function. The current 
layer ARN is 
`arn:aws:lambda:us-east-1:466768951184:layer:newrelic-lambda-extension:8`

TODO: Fix the ARN above

You'll also need to make the New Relic license key available to the extension. Use the `newrelic-lambda`
CLI to install the managed secret, and then add the permission for the secret to your Lambda execution role.


## Getting Started
>[Simple steps to start working with the software similar to a "Hello World"]

TODO: do we need this section, or is installation enough?

## Building

Use the included `Makefile` to compile the extension. 

    make dist
    
will create the extension binary in `./extensions/newrelic-lambda-extension`

## Testing

To test locally, acquire the AWS extension test harness first. Then:

TODO: Link to the AWS SDK that has the test harness

1. (Optional) Use the `newrelic-lambda` CLI to create the license key managed secret in your AWS account and region.
2. Build the docker container for sample function code. Give it the tag `lambda_ext`.
   - Be sure to include your lambda function in the container.
3. Start up your container.

   - Using AWS Secret Manager
   
            export AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id --profile default)
            export AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key --profile default)
            export AWS_SESSION_TOKEN=$(aws configure get aws_session_token --profile default)
    
            docker run --rm -v $(pwd)/extensions:/opt/extensions -p 9001:8080 \
                -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_SESSION_TOKEN \
                lambda_ext:latest \
                -h function.handler -c '{}' -t 60000

   - Or, setting the license key directly
   
            docker run --rm \
                -v $(pwd)/extensions:/opt/extensions \
                -p 9001:8080 \
                lambda_ext:latest \
                -h function.handler -c '{"NEW_RELIC_LICENSE_KEY": "your-license-key-here"}' -t 60000

4. To invoke the sample lambda run: 
   
       curl -XPOST 'http://localhost:9001/2015-03-31/functions/function.handler/invocations' \
           -d 'invoke-payload'

5. Finally, you can exercise the container shutdown lifecycle event with:

        curl -XPOST 'http://localhost:9001/test/shutdown' \
            -d '{"timeoutMs": 5000 }'

## Support

New Relic hosts and moderates an online forum where customers can interact with New Relic employees as well as other customers to get help and share best practices. Like all official New Relic open source projects, there's a related Community topic in the New Relic Explorers Hub. You can find this project's topic/threads here:

>Add the url for the support thread here

TODO: add the URL

## Contributing
We encourage your contributions to improve `newrelic-lambda-extension`! Keep in mind when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.
If you have any questions, or to execute our corporate CLA, required if your contribution is on behalf of a company,  please drop us an email at opensource@newrelic.com.

## License
`newrelic-lambda-extension` is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.
The `newrelic-lambda-extension` also uses source code from third-party libraries. You can find full details on which libraries are used and the terms under which they are licensed in the third-party notices document.
