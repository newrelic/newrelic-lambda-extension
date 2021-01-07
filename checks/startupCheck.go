package checks

import (
	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type checkFn func(*config.Configuration, *api.RegistrationResponse) error

/// Register checks here
var checks = []checkFn {
	exampleCheckFunction,
}

func RunChecks(conf *config.Configuration, reg *api.RegistrationResponse) {
	for _, check := range checks {
		err := check(conf, reg)
		if err != nil {
			util.Logln("Startup check failed", err)
			//TODO: send something to NR as well
		}
	}
}

func exampleCheckFunction(*config.Configuration, *api.RegistrationResponse) error {
	return nil
}
