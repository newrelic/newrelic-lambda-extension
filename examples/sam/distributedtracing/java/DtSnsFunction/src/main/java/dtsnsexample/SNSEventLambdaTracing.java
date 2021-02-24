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

public class SNSEventLambdaTracing<O> extends LambdaTracing<SNSEvent, O> {
    @Override
    protected SpanContext extractContext(Tracer tracer, Object input) {
        if (input instanceof SNSEvent) {
            SNSEvent sqsEvent = (SNSEvent) input;
            final String traceContextJson = sqsEvent.getRecords()
                    .get(0)
                    .getSNS()
                    .getMessageAttributes()
                    .get("NRDT")
                    .getValue();
            try {
                @SuppressWarnings("unchecked")
                final Map<String, String> parsedJson = (Map<String, String>) new JSONParser().parse(traceContextJson);
                TextMapAdapter carrier = new TextMapAdapter(parsedJson);
                return tracer.extract(Format.Builtin.HTTP_HEADERS, carrier);
            } catch (ParseException | ClassCastException | IllegalArgumentException e) {
                Log.getInstance().out(e.toString());
                Log.getInstance().out(traceContextJson);
            }
        }
        return null;
    }
}
