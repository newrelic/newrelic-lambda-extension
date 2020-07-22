# lambda-extension-exploration-golang

## Running instructions
1. Pull down s3 object containing source code for extensionsbetashare
2. `env GOOS=linux GOARCH=amd64 go build` to build executable that can run on linux

### Pulled from extensionsbetashare docs:
3. build the docker container for sample function code
4. start up your contianer - `docker run --rm -v ${path to executable}:/opt/extensions -p 9001:8080 {docker image} -h function.handler -c '{"FOO": "BAR"}' -t 60000`

From here you should see log lines indicating that start up and registration was successful

5. To invoke the sample lambda run: `curl -XPOST "http://localhost:9001/2015-03-31/functions/function.handler/invocations" -d 'invoke-payload'`

You should see a counter increment as well as an INVOKE event payload
