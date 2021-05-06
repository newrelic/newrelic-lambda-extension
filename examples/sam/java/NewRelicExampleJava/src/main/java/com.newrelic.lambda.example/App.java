package com.newrelic.lambda.example;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;
import io.opentracing.Scope;
import io.opentracing.Span;
import io.opentracing.Tracer;
import io.opentracing.util.GlobalTracer;

import java.util.Map;

public class App implements RequestHandler<Map<String, Object>, String> {

    public String handleRequest(final Map<String, Object> input, final Context context) {
        final Tracer tracer = GlobalTracer.get();

        // This is an example of a custom span. `FROM Span SELECT * WHERE name='MyJavaSpan'` in New Relic will find this event.
        Span customSpan = tracer.buildSpan("MyJavaSpan").start();
        try (Scope scope = tracer.activateSpan(customSpan)) {
            // Here, we add a tag to our custom span
            customSpan.setTag("zip", "zap");
        } finally {
            customSpan.finish();
        }

        // This tag gets added to the function invocation's root span, since it's active.
        tracer.activeSpan().setTag("customAttribute", "customAttributeValue");

        // As normal, anything you write to stdout ends up in CloudWatch
        System.out.println("Hello, world");

        return "Success!";
    }
}
