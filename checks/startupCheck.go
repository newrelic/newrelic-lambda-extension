package checks

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type checkFn func(*config.Configuration, *api.RegistrationResponse) error

type LogSender interface {
	SendFunctionLogs(lines []logserver.LogLine) error
}

/// Register checks here
var checks = []checkFn{
	exampleCheckFunction,
	runtimeHandlerWrapperCheck,
}

func RunChecks(conf *config.Configuration, reg *api.RegistrationResponse, logSender LogSender) {
	for _, check := range checks {
		_ = runCheck(conf, reg, logSender, check)
	}
}

func runCheck(conf *config.Configuration, reg *api.RegistrationResponse, logSender LogSender, check checkFn) error {
	err := check(conf, reg)
	if err != nil {
		errLog := fmt.Sprintf("Startup check failed: %v", err)
		util.Logln(errLog)

		//Send a log line to NR as well
		_ = logSender.SendFunctionLogs([]logserver.LogLine{
			{
				Time:      time.Now(),
				RequestID: "0",
				Content:   []byte(errLog),
			},
		})
	}
	return err
}

func exampleCheckFunction(*config.Configuration, *api.RegistrationResponse) error {
	return nil
}

// runtimeHandlerWrapperCheck checks that the _HANDLER is set correctly in the event
// that the user has set the NEW_RELIC_LAMBDA_HANDLER environment variable. Also checks
// that the user is using a supported runtime when setting this environment variable.
func runtimeHandlerWrapperCheck(c *config.Configuration, r *api.RegistrationResponse) error {
	newRelicLambdaHandler := os.Getenv("NEW_RELIC_LAMBDA_HANDLER")

	// If user is not using NEW_RELIC_LAMBDA_HANDLER then it is assumed they are
	// manually wrapping their function
	if newRelicLambdaHandler == "" {
		return nil
	}

	runtime := util.DetectRuntime()

	if strings.HasPrefix(runtime, "nodejs") && r.Handler != "newrelic-lambda-wrapper.handler" {
		return fmt.Errorf("Handler must be 'newrelic-lambda-wrapper.handler' when NEW_RELIC_LAMBDA_HANDLER environment variable set, is currently '%s'", r.Handler)
	}

	if strings.HasPrefix(runtime, "python") && r.Handler != "newrelic_lambda_wrapper.handler" {
		return fmt.Errorf("Handler must be 'newrelic_lambda_wrapper.handler' when NEW_RELIC_LAMBDA_HANDLER environment variable set, is currently '%s'", r.Handler)
	}

	if !strings.HasPrefix(runtime, "nodejs") && !strings.HasPrefix(runtime, "python") {
		return fmt.Errorf("Runtime '%s' unsupported for NEW_RELIC_LAMBDA_HANDLER environment variablebased wrapping, unset this and set your handler to '%s'", runtime, newRelicLambdaHandler)
	}

	return nil
}
