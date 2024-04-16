package checks

import (
	"context"
	"fmt"
	"time"

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
func sanityCheck(ctx context.Context, conf *config.Configuration, res *api.RegistrationResponse, _ runtimeConfig) error {
	if util.AnyEnvVarsExist(awsLogIngestionEnvVars) {
		return fmt.Errorf("Environment variable '%s' is used by aws-log-ingestion and has no effect here. Recommend unsetting this environment variable within this function.", util.AnyEnvVarsExistString(awsLogIngestionEnvVars))
	}

	envKeyExists := util.EnvVarExists("NEW_RELIC_LICENSE_KEY")
	var timeout = 1 * time.Second
	ctxSecret, cancelSecret := context.WithTimeout(ctx, timeout)
	defer cancelSecret()
	isSecretConfigured := credentials.IsSecretConfigured(ctxSecret, conf)

	ctxSSMParameter, cancelSSMParameter := context.WithTimeout(ctx, timeout)
	defer cancelSSMParameter()

	isSSMParameterConfigured := false
	if conf.LicenseKeySSMParameterName != "" {
		isSSMParameterConfigured = credentials.IsSSMParameterConfigured(ctxSSMParameter, conf)
	}
	

	if isSecretConfigured && envKeyExists {
		return fmt.Errorf("There is both a AWS Secrets Manager secret and a NEW_RELIC_LICENSE_KEY environment variable set. Recommend removing the NEW_RELIC_LICENSE_KEY environment variable and using the AWS Secrets Manager secret.")
	}

	if isSSMParameterConfigured && envKeyExists {
		return fmt.Errorf("There is both a AWS Parameter Store parameter and a NEW_RELIC_LICENSE_KEY environment variable set. Recommend removing the NEW_RELIC_LICENSE_KEY environment variable and using the AWS Parameter Store parameter.")
	}

	if isSecretConfigured && isSSMParameterConfigured {
		return fmt.Errorf("There is both a AWS Secrets Manager secret and a AWS Parameter Store parameter set. Recommend using just one.")
	}

	return nil
}
