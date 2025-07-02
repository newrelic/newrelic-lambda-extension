package checks

import (
	"context"
	"fmt"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type checkFn func(context.Context, *config.Configuration, *api.RegistrationResponse, runtimeConfig) error

type LogSender interface {
	SendFunctionLogs(ctx context.Context, invokedFunctionARN string, lines []logserver.LogLine, entityGuid string) error
}

/// Register checks here
var checks = map[string]checkFn{
    "agent":         agentVersionCheck, 
    "handler":       handlerCheck,
    "sanity":        sanityCheck, 
    "vendor":        vendorCheck, 
}

func RunChecks(ctx context.Context, conf *config.Configuration, reg *api.RegistrationResponse, logSender LogSender) {
	runtimeConfig, err := checkAndReturnRuntime()
	if err != nil {
		errLog := fmt.Sprintf("There was an issue querying for the latest agent version: %v", err)
		util.Logln(errLog)
	}

	for checkName, check := range checks {
		if conf.IgnoreExtensionChecks[checkName] {
			continue
		}
		runCheck(ctx, conf, reg, runtimeConfig, logSender, check)
	}
}

func runCheck(ctx context.Context, conf *config.Configuration, reg *api.RegistrationResponse, r runtimeConfig, logSender LogSender, check checkFn) error {
	err := check(ctx, conf, reg, r)
	if err != nil {
		errLog := fmt.Sprintf("Startup check warning: %v", err)
		util.Logln(errLog)
		var entityGuid string
		//Send a log line to NR as well
		logSender.SendFunctionLogs(ctx, "", []logserver.LogLine{
			{
				Time:      time.Now(),
				RequestID: "0",
				Content:   []byte(errLog),
			},
		},
		entityGuid)
	}

	return err
}
