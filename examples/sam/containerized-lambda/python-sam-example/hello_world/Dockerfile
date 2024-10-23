FROM public.ecr.aws/newrelic-lambda-layers-for-docker/newrelic-lambda-layers-python:312 AS layer

FROM public.ecr.aws/lambda/python:3.12

COPY --from=layer /opt/ /opt/

COPY app.py requirements.txt ./

RUN python3.12 -m pip install -r requirements.txt -t .

CMD [ "newrelic_lambda_wrapper.handler" ]
