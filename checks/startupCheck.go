package checks

import (
	"fmt"
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
	checkHandler,
	sanityCheck,
	vendorCheck,
}

func RunChecks(conf *config.Configuration, reg *api.RegistrationResponse, logSender LogSender) {
	for _, check := range checks {
		runCheck(conf, reg, logSender, check)
	}
}

func runCheck(conf *config.Configuration, reg *api.RegistrationResponse, logSender LogSender, check checkFn) error {
	err := check(conf, reg)

	if err != nil {
		errLog := fmt.Sprintf("Startup check failed: %v", err)
		util.Logln(errLog)

		//Send a log line to NR as well
		logSender.SendFunctionLogs([]logserver.LogLine{
			{
				Time:      time.Now(),
				RequestID: "0",
				Content:   []byte(errLog),
			},
		})
	}

	return err
}
