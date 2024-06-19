package checks

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/credentials"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
)

type mockSecretManager struct {
	secretsmanageriface.SecretsManagerAPI
	validSecrets []string
}

func (m mockSecretManager) GetSecretValueWithContext(_ context.Context, input *secretsmanager.GetSecretValueInput, _ ...request.Option) (*secretsmanager.GetSecretValueOutput, error) {
	for _, secret := range m.validSecrets {
		if secret == *input.SecretId {
			return &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(`{"LicenseKey": "foo"}`),
			}, nil
		}
	}

	return nil, fmt.Errorf("Secret not found")
}

type mockSSM struct {
	ssmiface.SSMAPI
	validParameters   []string
	IsParameterCalled bool
}

func (m *mockSSM) GetParameterWithContext(_ context.Context, input *ssm.GetParameterInput, _ ...request.Option) (*ssm.GetParameterOutput, error) {
	m.IsParameterCalled = true
	for _, parameter := range m.validParameters {
		if parameter == *input.Name {
			return &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: aws.String("bar"),
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("Parameter not found")
}

func TestSanityCheck(t *testing.T) {
	table := []struct {
		Name string

		Conf           config.Configuration
		Environment    map[string]string
		SecretsManager secretsmanageriface.SecretsManagerAPI
		SSM            ssmiface.SSMAPI

		ExpectedErr string
	}{
		{
			Name: "returns error when nothing is configured",

			Conf:           config.Configuration{},
			Environment:    map[string]string{},
			SecretsManager: mockSecretManager{},
			SSM:            &mockSSM{},

			ExpectedErr: "No configured license key found, attempting fallback to default AWS Secrets Manager secret with NEW_RELIC_LICENSE_KEY.",
		},
		{
			Name: "returns nil when just the environment variable exists",

			Conf: config.Configuration{},
			Environment: map[string]string{
				"NEW_RELIC_LICENSE_KEY": "12345",
			},
			SecretsManager: mockSecretManager{},
			SSM:            &mockSSM{},
		},
		{
			Name: "return nil when just the secret is configured",

			Conf: config.Configuration{
				LicenseKeySecretId: "secret",
			},
			Environment: map[string]string{},
			SecretsManager: mockSecretManager{
				validSecrets: []string{"secret"},
			},
			SSM: &mockSSM{},
		},
		{
			Name: "return nil when just the parameter is configured",

			Conf: config.Configuration{
				LicenseKeySSMParameterName: "parameter",
			},
			Environment:    map[string]string{},
			SecretsManager: mockSecretManager{},
			SSM: &mockSSM{
				validParameters: []string{"parameter"},
			},
		},
		{
			Name: "returns error when log ingestion environment variable is set",

			Conf: config.Configuration{},
			Environment: map[string]string{
				"DEBUG_LOGGING_ENABLED": "1",
			},
			SecretsManager: mockSecretManager{},
			SSM:            &mockSSM{},

			ExpectedErr: "Environment variable 'DEBUG_LOGGING_ENABLED' is used by aws-log-ingestion and has no effect here. Recommend unsetting this environment variable within this function.",
		},
		{
			Name: "returns error when environment variable and secret are configured",

			Conf: config.Configuration{
				LicenseKeySecretId: "secret",
			},
			Environment: map[string]string{
				"NEW_RELIC_LICENSE_KEY": "12345",
			},
			SecretsManager: mockSecretManager{
				validSecrets: []string{"secret"},
			},
			SSM: &mockSSM{},

			ExpectedErr: "There is both a AWS Secrets Manager secret and a NEW_RELIC_LICENSE_KEY environment variable set. Recommend removing the NEW_RELIC_LICENSE_KEY environment variable and using the AWS Secrets Manager secret.",
		},
		{
			Name: "returns error when environment variable and parameter are configured",

			Conf: config.Configuration{
				LicenseKeySSMParameterName: "parameter",
			},
			Environment: map[string]string{
				"NEW_RELIC_LICENSE_KEY": "12345",
			},
			SecretsManager: mockSecretManager{},
			SSM: &mockSSM{
				validParameters: []string{"parameter"},
			},

			ExpectedErr: "There is both a AWS Parameter Store parameter and a NEW_RELIC_LICENSE_KEY environment variable set. Recommend removing the NEW_RELIC_LICENSE_KEY environment variable and using the AWS Parameter Store parameter.",
		},
		{
			Name: "returns error when secret and parameter are configured",
			Conf: config.Configuration{
				LicenseKeySecretId:         "secret",
				LicenseKeySSMParameterName: "parameter",
			},
			Environment: map[string]string{},
			SecretsManager: mockSecretManager{
				validSecrets: []string{"secret"},
			},
			SSM: &mockSSM{
				validParameters: []string{"parameter"},
			},

			ExpectedErr: "There is both a AWS Secrets Manager secret and a AWS Parameter Store parameter set. Recommend using just one.",
		},
	}

	ctx := context.Background()

	for _, entry := range table {
		t.Run(entry.Name, func(t *testing.T) {
			credentials.OverrideSecretsManager(entry.SecretsManager)
			credentials.OverrideSSM(entry.SSM)

			for name, value := range entry.Environment {
				os.Setenv(name, value)
			}

			err := sanityCheck(ctx, &entry.Conf, &api.RegistrationResponse{}, runtimeConfig{})

			if entry.ExpectedErr != "" {
				assert.EqualError(t, err, entry.ExpectedErr)
			} else {
				assert.Nil(t, err)
			}

			for name := range entry.Environment {
				os.Unsetenv(name)
			}
		})
	}
}

func TestSanityCheckSSMParameter(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		ssmParameterName  string
		validParameters   []string
		expectParamCalled bool
		expectedErr       error
	}{
		{
			name:              "SSM Parameter configured",
			ssmParameterName:  "parameter",
			validParameters:   []string{"parameter"},
			expectParamCalled: true,
			expectedErr:       nil,
		},
		{
			name:              "SSM Parameter not configured",
			expectParamCalled: false,
			expectedErr:       nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conf := config.Configuration{
				LicenseKeySSMParameterName: tc.ssmParameterName,
			}

			mSSM := &mockSSM{
				validParameters: tc.validParameters,
			}

			credentials.OverrideSSM(mSSM)

			err := sanityCheck(ctx, &conf, &api.RegistrationResponse{}, runtimeConfig{})

			assert.Equal(t, tc.expectedErr, err, "Error from sanityCheck")
			assert.Equal(t, tc.expectParamCalled, mSSM.IsParameterCalled, "Error in expected SSM parameter check")
		})
	}
}
