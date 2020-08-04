# newrelic-lambda-extension

## Running instructions

1. Pull down s3 object containing source code for extensionsbetashare.
2. To build the executable: If running locally `make build`. If deploying to AWS Lambda environment or running Docker container below `make dist`.

### Pulled from extensionsbetashare docs:

3. build the docker container for sample function code. Give it the tag `lambda_ext`.
4. start up your container.
 
        docker run --rm -v $(pwd)/extensions:/opt/extensions -p 9001:8080 \
            lambda_ext:latest \
            -h function.handler -c '{}' -t 60000

From here you should see log lines indicating that start up and registration was successful.

5. To invoke the sample lambda run: 

        curl -XPOST 'http://localhost:9001/2015-03-31/functions/function.handler/invocations' \
            -d 'invoke-payload'

    You should see a counter increment as well as an INVOKE event payload.

6. Finally, you can exercise the container shutdown lifecycle event with:

        curl -XPOST 'http://localhost:9001/test/shutdown' \
            -d '{"timeoutMs": 5000 }'
