package checks

import (
	"fmt"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/credentials"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

var (
	awsLogIngestionEnvVars = []string{
		"DEBUG_LOGGING_ENABLED",
		"INFRA_ENABLED",
		"LICENSE_KEY",
		"LOGGING_ENABLED",
		"NR_INFRA_ENDPOINT",
		"NR_LOGGING_ENDPOINT",
	}
)

// sanityCheck checks for configuration that is either misplaced or in conflict
func sanityCheck(conf *config.Configuration, res *api.RegistrationResponse, _ runtimeConfig) error {
	if util.AnyEnvVarsExist(awsLogIngestionEnvVars) {
		return fmt.Errorf("Environment varaible '%s' is used by aws-log-ingestion and has no effect here. Recommend unsetting this environment variable within this function.", util.AnyEnvVarsExistString(awsLogIngestionEnvVars))
	}

	if credentials.IsSecretConfigured(conf) && util.EnvVarExists("NEW_RELIC_LICENSE_KEY") {
		return fmt.Errorf("There is both a AWS Secrets Manager secret and a NEW_RELIC_LICENSE_KEY environment variable set. Recommend removing the NEW_RELIC_LICENSE_KEY environment variable and using the AWS Secrets Manager secret.")
	}

	return nil
}
