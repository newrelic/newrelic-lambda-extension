package credentials

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/stretchr/testify/assert"
)

func TestGetLicenseKeySecretId(t *testing.T) {
	secretId := getLicenseKeySecretId(&config.Configuration{})
	assert.Equal(t, defaultSecretId, secretId)

	var testSecretId = "testSecretName"
	var conf = &config.Configuration{LicenseKeySecretId: testSecretId}
	secretId = getLicenseKeySecretId(conf)
	assert.Equal(t, testSecretId, secretId)
}

func TestGetLicenseKeySSMParameterName(t *testing.T) {
	parameterName := getLicenseKeySSMParameterName(&config.Configuration{})
	assert.Equal(t, defaultSecretId, parameterName)

	var testParameterName = "testParameterName"
	var conf = &config.Configuration{LicenseKeySSMParameterName: testParameterName}
	parameterName = getLicenseKeySSMParameterName(conf)
	assert.Equal(t, testParameterName, parameterName)
}

type mockSecretManager struct {
	secretsmanageriface.SecretsManagerAPI
	validSecrets []string
}

const mockSecretManagerKeyValue = "licenseKeyStoredAsSecret"

func (m mockSecretManager) GetSecretValueWithContext(_ context.Context, input *secretsmanager.GetSecretValueInput, _ ...request.Option) (*secretsmanager.GetSecretValueOutput, error) {
	for _, secret := range m.validSecrets {
		if secret == *input.SecretId {
			return &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(fmt.Sprintf(`{"LicenseKey": "%s"}`, mockSecretManagerKeyValue)),
			}, nil
		}
	}

	return nil, fmt.Errorf("Secret not found")
}

func TestIsSecretConfigured(t *testing.T) {
	ctx := context.Background()
	assert.False(t, IsSecretConfigured(ctx, &config.Configuration{}))

	OverrideSecretsManager(mockSecretManager{
		validSecrets: []string{"testSecretName"},
	})
	assert.True(t, IsSecretConfigured(ctx, &config.Configuration{
		LicenseKeySecretId: "testSecretName",
	}))

	OverrideSecretsManager(mockSecretManager{})
	assert.False(t, IsSecretConfigured(ctx, &config.Configuration{
		LicenseKeySecretId: "testSecretName",
	}))
}

type mockSSM struct {
	ssmiface.SSMAPI
	validParameters []string
}

const mockParameterStoreKeyValue = "licenseKeyStoredAsParameter"

func (m mockSSM) GetParameterWithContext(_ context.Context, input *ssm.GetParameterInput, _ ...request.Option) (*ssm.GetParameterOutput, error) {
	for _, parameter := range m.validParameters {
		if parameter == *input.Name {
			return &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: aws.String(mockParameterStoreKeyValue),
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("Parameter not found")
}

func TestIsSSMParameterConfigured(t *testing.T) {
	ctx := context.Background()
	assert.False(t, IsSSMParameterConfigured(ctx, &config.Configuration{}))

	OverrideSSM(mockSSM{
		validParameters: []string{"testParameterName"},
	})
	assert.True(t, IsSSMParameterConfigured(ctx, &config.Configuration{
		LicenseKeySSMParameterName: "testParameterName",
	}))

	OverrideSSM(mockSSM{})
	assert.False(t, IsSSMParameterConfigured(ctx, &config.Configuration{
		LicenseKeySSMParameterName: "testParameterName",
	}))
}

func TestGetNewRelicLicenseKey(t *testing.T) {
	table := []struct {
		Name string

		Conf           config.Configuration
		Environment    map[string]string
		SecretsManager secretsmanageriface.SecretsManagerAPI
		SSM            ssmiface.SSMAPI

		ExpectedKey string
		ExpectedErr string
	}{
		{
			Name: "uses config license key when present",
			Conf: config.Configuration{
				LicenseKey: "in_config",
			},

			ExpectedKey: "in_config",
		},
		{
			Name: "loads license key from secret when configured",
			Conf: config.Configuration{
				LicenseKeySecretId: "testSecretName",
			},
			SecretsManager: mockSecretManager{
				validSecrets: []string{"testSecretName"},
			},

			ExpectedKey: mockSecretManagerKeyValue,
		},
		{
			Name: "loads license key from parameter when configured",
			Conf: config.Configuration{
				LicenseKeySSMParameterName: "testParameterName",
			},
			SSM: mockSSM{
				validParameters: []string{"testParameterName"},
			},

			ExpectedKey: mockParameterStoreKeyValue,
		},
		{
			Name: "loads license key from environment variable if not configured",
			Conf: config.Configuration{},
			Environment: map[string]string{
				"NEW_RELIC_LICENSE_KEY": "12345",
			},

			ExpectedKey: "12345",
		},
		{
			Name: "returns error if secret is configured but unavailable",
			Conf: config.Configuration{
				LicenseKeySecretId: "testSecretName",
			},
			SecretsManager: mockSecretManager{},

			ExpectedErr: "Secret not found",
		},
		{
			Name: "returns error if parameter is configured but unavailable",
			Conf: config.Configuration{
				LicenseKeySSMParameterName: "testParameterName",
			},
			SSM: mockSSM{},

			ExpectedErr: "Parameter not found",
		},
		{
			Name: "defaults to license key",
			Conf: config.Configuration{
				LicenseKey:         "12345",
				LicenseKeySecretId: "testSecretName",
			},
			SecretsManager: mockSecretManager{},
			ExpectedKey:    "12345",
		},
		{
			Name:           "returns error if no license key is configured",
			Conf:           config.Configuration{},
			SecretsManager: mockSecretManager{},
			SSM:            mockSSM{},

			ExpectedErr: "No license key configured",
		},
		{
			Name: "loads license key from fallback secret",
			Conf: config.Configuration{},
			SecretsManager: mockSecretManager{
				validSecrets: []string{"NEW_RELIC_LICENSE_KEY"},
			},
			SSM: mockSSM{},

			ExpectedKey: mockSecretManagerKeyValue,
		},
		{
			Name:           "loads license key from fallback parameter",
			Conf:           config.Configuration{},
			SecretsManager: mockSecretManager{},
			SSM: mockSSM{
				validParameters: []string{"NEW_RELIC_LICENSE_KEY"},
			},

			ExpectedKey: mockParameterStoreKeyValue,
		},
	}

	ctx := context.Background()

	for _, entry := range table {
		t.Run(entry.Name, func(t *testing.T) {
			OverrideSecretsManager(entry.SecretsManager)
			OverrideSSM(entry.SSM)

			for name, value := range entry.Environment {
				os.Setenv(name, value)
			}

			lk, err := GetNewRelicLicenseKey(ctx, &entry.Conf)

			if entry.ExpectedErr == "" {
				assert.Equal(t, entry.ExpectedKey, lk)
				assert.NoError(t, err)
			} else {
				assert.Empty(t, lk)
				assert.EqualError(t, err, entry.ExpectedErr)
			}

			for name := range entry.Environment {
				os.Unsetenv(name)
			}
		})
	}
}

func TestDecodeLicenseKey(t *testing.T) {
	invalidJson := "invalid json"
	decoded, err := decodeLicenseKey(&invalidJson)
	assert.Empty(t, decoded)
	assert.Error(t, err)
}

func TestDecodeLicenseKeyValidButWrong(t *testing.T) {
	badJson := "{\"some\": \"garbage\"}"
	decoded, err := decodeLicenseKey(&badJson)
	assert.Empty(t, decoded)
	assert.Error(t, err)
}
