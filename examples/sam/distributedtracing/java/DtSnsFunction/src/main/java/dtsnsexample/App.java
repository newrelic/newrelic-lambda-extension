package dtsnsexample;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;
import com.amazonaws.services.lambda.runtime.events.SNSEvent;
import com.newrelic.opentracing.LambdaTracer;
import io.opentracing.util.GlobalTracer;

/**
 * Handler for requests to Lambda function.
 */
public class App implements RequestHandler<SNSEvent, Object> {
    static {
        // Register the New Relic OpenTracing LambdaTracer as the Global Tracer
        GlobalTracer.registerIfAbsent(LambdaTracer.INSTANCE);
    }

    public Object handleRequest(final SNSEvent snsEvent, final Context context) {
        // Note the use of a custom LambdaTracing subclass.
        return new SNSEventLambdaTracing<>().instrumentRequest(snsEvent, context, this::handleInvocation);
    }

    public Object handleInvocation(final SNSEvent snsEvent, final Context context) {
        for (SNSEvent.SNSRecord r : snsEvent.getRecords()) {
            final SNSEvent.SNS sns = r.getSNS();
            final String message = sns.getMessage();
            context.getLogger().log(message);
            GlobalTracer.get().activeSpan().setTag("snsMessage", message);
        }

        return null;
    }
}
