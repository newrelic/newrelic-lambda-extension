package dtsnsexample;

import com.amazonaws.services.lambda.runtime.events.SNSEvent;
import com.newrelic.opentracing.aws.LambdaTracing;
import com.newrelic.opentracing.logging.Log;
import io.opentracing.SpanContext;
import io.opentracing.Tracer;
import io.opentracing.propagation.Format;
import io.opentracing.propagation.TextMapAdapter;
import org.json.simple.parser.JSONParser;
import org.json.simple.parser.ParseException;

import java.util.Map;

/**
 * This class extends LambdaTracing to specialize in SNSEvent messages, allowing us to recover
 * the trace context stored in the message attributes by the sender. It would also be possible to encode
 * the trace context with one attribute per key, rather than in a single attribute JSON string, as we do
 * in this example.
 *
 * @param <R> The result type for the handler function.
 */
public class SNSEventLambdaTracing<R> extends LambdaTracing<SNSEvent, R> {
    @Override
    protected SpanContext extractContext(Tracer tracer, Object input) {
        if (input instanceof SNSEvent) {
            SNSEvent sqsEvent = (SNSEvent) input;

            // Fetch the NRDT attribute value from the (first) message.
            final String traceContextJson = sqsEvent.getRecords()
                    .get(0)
                    .getSNS()
                    .getMessageAttributes()
                    .getOrDefault("NRDT", new SNSEvent.MessageAttribute())
                    .getValue();
            if (traceContextJson != null) {
                try {
                    // Here, JSONParser will produce a JSONObject, which implements the Map interface.
                    @SuppressWarnings("unchecked") final Map<String, String> parsedJson = (Map<String, String>) new JSONParser().parse(traceContextJson);
                    // TextMapAdapter wraps a Map and exposes its entries for trace context extraction
                    TextMapAdapter carrier = new TextMapAdapter(parsedJson);
                    // At this point, the trace context values look like they would in the headers of an HTTP request,
                    // so that's how we choose to treat them: the value part of the "newrelic" map entry is base64-encoded JSON.
                    return tracer.extract(Format.Builtin.HTTP_HEADERS, carrier);
                } catch (ParseException | ClassCastException | IllegalArgumentException e) {
                    Log.getInstance().out("Failed to extract trace context: " + e.toString());
                    Log.getInstance().out(traceContextJson);
                }
            }
        }
        return null;
    }
}
