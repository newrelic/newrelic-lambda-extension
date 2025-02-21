package main

import (
	"context"
	"fmt"

	"github.com/newrelic/go-agent/v3/integrations/nrlambda"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func handler(ctx context.Context) (string, error) {
	// At this point, we're handling an invocation. Cold start is over; this code runs for each invocation.
	// We'd like to add a custom event, and a custom attribute. For that, we need the transaction.
	if txn := newrelic.FromContext(ctx); nil != txn {
		// This is an example of a custom event. `FROM MyGoEvent SELECT *` in New Relic will find this event.
		txn.Application().RecordCustomEvent("MyGoEvent", map[string]interface{}{
			"zip": "zap",
		})

		// This attribute gets added to the normal AwsLambdaInvocation event
		txn.AddAttribute("customAttribute", "customAttributeValue")
	}
	// As normal, anything you write to stdout ends up in CloudWatch
	fmt.Println("hello world!")

	return "Success!", nil
}

func main() {
	// Here we are in cold start. Anything you do in main happens once.
	// In main, we initialize the agent.
	app, err := newrelic.NewApplication(nrlambda.ConfigOption())
	if nil != err {
		fmt.Println("error creating app (invalid config):", err)
	}
	// Then we start the lambda handler using `nrlambda` rather than `lambda`
	nrlambda.Start(handler, app)
}
