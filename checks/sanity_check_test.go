package checks

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
	"github.com/stretchr/testify/assert"
)

type mockSecretManager struct {
	secretsmanageriface.SecretsManagerAPI
}

func (mockSecretManager) GetSecretValue(*secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(`{"LicenseKey": "foo"}`),
	}, nil
}

func TestSanityCheck(t *testing.T) {
	if util.AnyEnvVarsExist(awsLogIngestionEnvVars) {
		assert.Error(t, sanityCheck(&config.Configuration{}, &api.RegistrationResponse{}, runtimeConfig{}))
	} else {
		assert.Nil(t, sanityCheck(&config.Configuration{}, &api.RegistrationResponse{}, runtimeConfig{}))
	}

	os.Setenv("DEBUG_LOGGING_ENABLED", "1")
	assert.Error(t, sanityCheck(&config.Configuration{}, &api.RegistrationResponse{}, runtimeConfig{}))
	os.Unsetenv("DEBUG_LOGGING_ENABLED")

	os.Setenv("NEW_RELIC_LICENSE_KEY", "foobar")
	secrets = &mockSecretManager{}
	assert.Error(t, sanityCheck(&config.Configuration{}, &api.RegistrationResponse{}, runtimeConfig{}))
	os.Unsetenv("NEW_RELIC_LICENSE_KEY")
}
