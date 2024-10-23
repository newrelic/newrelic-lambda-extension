# Define the New Relic pre-built image
FROM public.ecr.aws/newrelic-lambda-layers-for-docker/newrelic-lambda-layers-java:17 AS layer

FROM public.ecr.aws/sam/build-java17:latest as build-image


WORKDIR "/task"
COPY src/ src/
COPY pom.xml ./

RUN mvn -q clean install
RUN mvn dependency:copy-dependencies -DincludeScope=compile


FROM public.ecr.aws/lambda/java:17

COPY --from=layer /opt/ /opt/
COPY --from=build-image /task/target/classes /var/task/
COPY --from=build-image /task/target/dependency /var/task/lib

# Command can be overwritten by providing a different command in the template directly.
CMD [ "com.newrelic.java.HandlerWrapper::handleRequest" ]
