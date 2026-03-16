# Define the New Relic pre-built image
FROM public.ecr.aws/newrelic-lambda-layers-for-docker/newrelic-lambda-layers-nodejs:20 AS layer

FROM public.ecr.aws/lambda/nodejs:20

# Copy New Relic Layer code
COPY --from=layer /opt/ /opt/

COPY app.mjs package*.json ./

RUN npm install

# CMD override to New Relic's handler wrapper
# For CJS use this CMD [ "newrelic-lambda-wrapper.handler" ]
CMD [ "/opt/nodejs/node_modules/newrelic-esm-lambda-wrapper/index.handler" ]