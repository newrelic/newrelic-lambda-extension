package dtsnsexample;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;
import com.amazonaws.services.lambda.runtime.events.SNSEvent;

import com.newrelic.opentracing.LambdaTracer;
import com.newrelic.opentracing.aws.LambdaTracing;
import io.opentracing.Scope;
import io.opentracing.Span;
import io.opentracing.Tracer;
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
        return LambdaTracing.instrument(snsEvent, context, this::handleInvocation);
    }


    public Object handleInvocation(final SNSEvent snsEvent, final Context context) {
        for (SNSEvent.SNSRecord r : snsEvent.getRecords()) {
            context.getLogger().log(r.getSNS().getMessage());
        }

        return null;
    }
}
