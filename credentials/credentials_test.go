package credentials

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
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
	validSecrets []string
}

const mockSecretManagerKeyValue = "licenseKeyStoredAsSecret"

func (m mockSecretManager) GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
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
	validParameters []string
}

const mockParameterStoreKeyValue = "licenseKeyStoredAsParameter"

func (m mockSSM) GetParameter(ctx context.Context, input *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	for _, parameter := range m.validParameters {
		if parameter == *input.Name {
			return &ssm.GetParameterOutput{
				Parameter: &types.Parameter{
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
		SecretsManager SecretsManagerAPI
		SSM            SSMAPI

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

func TestDecodeLicenseKeyValid(t *testing.T) {
	validJson := "{\"LicenseKey\": \"some_key\"}"
	decoded, err := decodeLicenseKey(&validJson)
	assert.Equal(t, "some_key", decoded)
	assert.NoError(t, err)
}

func TestDecodeLicenseKeyEmpty(t *testing.T) {
	emptyJson := "{\"LicenseKey\": \"\"}"
	decoded, err := decodeLicenseKey(&emptyJson)
	assert.Empty(t, decoded)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "malformed license key secret; missing \"LicenseKey\" attribute")
}

func TestDecodeLicenseKeyMissingField(t *testing.T) {
	missingFieldJson := "{\"NotLicenseKey\": \"some_value\"}"
	decoded, err := decodeLicenseKey(&missingFieldJson)
	assert.Empty(t, decoded)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "malformed license key secret; missing \"LicenseKey\" attribute")
}

func TestGetNewRelicLicenseKeyPriority(t *testing.T) {
	ctx := context.Background()

	originalSecrets := secretsAPI
	originalSSM := ssmAPI
	defer func() {
		secretsAPI = originalSecrets
		ssmAPI = originalSSM
	}()

	os.Setenv("NEW_RELIC_LICENSE_KEY", "env_value")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")

	OverrideSecretsManager(mockSecretManager{
		validSecrets: []string{"NEW_RELIC_LICENSE_KEY", "testSecretName"},
	})
	OverrideSSM(mockSSM{
		validParameters: []string{"NEW_RELIC_LICENSE_KEY", "testParameterName"},
	})

	conf := &config.Configuration{
		LicenseKey:                 "config_value",
		LicenseKeySecretId:         "testSecretName",
		LicenseKeySSMParameterName: "testParameterName",
	}

	lk, err := GetNewRelicLicenseKey(ctx, conf)
	assert.NoError(t, err)
	assert.Equal(t, "config_value", lk)
}

func TestGetNewRelicLicenseKeySecretPriority(t *testing.T) {
	ctx := context.Background()

	originalSecrets := secretsAPI
	originalSSM := ssmAPI
	defer func() {
		secretsAPI = originalSecrets
		ssmAPI = originalSSM
	}()

	os.Setenv("NEW_RELIC_LICENSE_KEY", "env_value")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")

	OverrideSecretsManager(mockSecretManager{
		validSecrets: []string{"NEW_RELIC_LICENSE_KEY", "testSecretName"},
	})
	OverrideSSM(mockSSM{
		validParameters: []string{"NEW_RELIC_LICENSE_KEY", "testParameterName"},
	})

	conf := &config.Configuration{
		LicenseKeySecretId:         "testSecretName",
		LicenseKeySSMParameterName: "testParameterName",
	}

	lk, err := GetNewRelicLicenseKey(ctx, conf)
	assert.NoError(t, err)
	assert.Equal(t, mockSecretManagerKeyValue, lk)
}

func TestGetNewRelicLicenseKeySSMParameterPriority(t *testing.T) {
	ctx := context.Background()

	originalSecrets := secretsAPI
	originalSSM := ssmAPI
	defer func() {
		secretsAPI = originalSecrets
		ssmAPI = originalSSM
	}()

	os.Setenv("NEW_RELIC_LICENSE_KEY", "env_value")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")

	OverrideSecretsManager(mockSecretManager{})
	OverrideSSM(mockSSM{
		validParameters: []string{"testParameterName"},
	})

	conf := &config.Configuration{
		LicenseKeySSMParameterName: "testParameterName",
	}

	lk, err := GetNewRelicLicenseKey(ctx, conf)
	assert.NoError(t, err)
	assert.Equal(t, mockParameterStoreKeyValue, lk)
}

func TestIsSecretConfiguredWithDefaultId(t *testing.T) {
	ctx := context.Background()

	originalSecrets := secretsAPI
	defer func() { secretsAPI = originalSecrets }()

	OverrideSecretsManager(mockSecretManager{
		validSecrets: []string{defaultSecretId},
	})

	assert.True(t, IsSecretConfigured(ctx, &config.Configuration{}))
}

func TestIsSSMParameterConfiguredWithDefaultParameter(t *testing.T) {
	ctx := context.Background()

	originalSSM := ssmAPI
	defer func() { ssmAPI = originalSSM }()

	OverrideSSM(mockSSM{
		validParameters: []string{defaultSecretId},
	})
	assert.True(t, IsSSMParameterConfigured(ctx, &config.Configuration{}))
}

func TestMockSecretManagerMultipleSecrets(t *testing.T) {
	ctx := context.Background()
	mock := mockSecretManager{
		validSecrets: []string{"secret1", "secret2", "secret3"},
	}

	input1 := &secretsmanager.GetSecretValueInput{SecretId: aws.String("secret1")}
	output1, err1 := mock.GetSecretValue(ctx, input1)
	assert.NoError(t, err1)
	assert.NotNil(t, output1.SecretString)

	input2 := &secretsmanager.GetSecretValueInput{SecretId: aws.String("secret2")}
	output2, err2 := mock.GetSecretValue(ctx, input2)
	assert.NoError(t, err2)
	assert.NotNil(t, output2.SecretString)

	input3 := &secretsmanager.GetSecretValueInput{SecretId: aws.String("nonexistent")}
	output3, err3 := mock.GetSecretValue(ctx, input3)
	assert.Error(t, err3)
	assert.Nil(t, output3)
	assert.Contains(t, err3.Error(), "Secret not found")
}

func TestMockSSMMultipleParameters(t *testing.T) {
	ctx := context.Background()
	mock := mockSSM{
		validParameters: []string{"param1", "param2", "param3"},
	}

	input1 := &ssm.GetParameterInput{Name: aws.String("param1")}
	output1, err1 := mock.GetParameter(ctx, input1)
	assert.NoError(t, err1)
	assert.NotNil(t, output1.Parameter)
	assert.Equal(t, mockParameterStoreKeyValue, *output1.Parameter.Value)

	input2 := &ssm.GetParameterInput{Name: aws.String("param2")}
	output2, err2 := mock.GetParameter(ctx, input2)
	assert.NoError(t, err2)
	assert.NotNil(t, output2.Parameter)

	input3 := &ssm.GetParameterInput{Name: aws.String("nonexistent")}
	output3, err3 := mock.GetParameter(ctx, input3)
	assert.Error(t, err3)
	assert.Nil(t, output3)
	assert.Contains(t, err3.Error(), "Parameter not found")
}

func TestOverrideFunctions(t *testing.T) {
	originalSecrets := secretsAPI
	originalSSM := ssmAPI
	defer func() {
		secretsAPI = originalSecrets
		ssmAPI = originalSSM
	}()

	mockSecretsManager := mockSecretManager{
		validSecrets: []string{"test-override-secret"},
	}
	OverrideSecretsManager(mockSecretsManager)

	ctx := context.Background()

	assert.True(t, IsSecretConfigured(ctx, &config.Configuration{
		LicenseKeySecretId: "test-override-secret",
	}))

	mockSSMClient := mockSSM{
		validParameters: []string{"test-override-parameter"},
	}
	OverrideSSM(mockSSMClient)

	assert.True(t, IsSSMParameterConfigured(ctx, &config.Configuration{
		LicenseKeySSMParameterName: "test-override-parameter",
	}))
}

func TestGetNewRelicLicenseKeyEnvironmentVariableFallback(t *testing.T) {
	ctx := context.Background()

	originalSecrets := secretsAPI
	originalSSM := ssmAPI
	defer func() {
		secretsAPI = originalSecrets
		ssmAPI = originalSSM
	}()

	os.Setenv("NEW_RELIC_LICENSE_KEY", "env_fallback_value")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")

	OverrideSecretsManager(mockSecretManager{})
	OverrideSSM(mockSSM{})

	conf := &config.Configuration{}

	lk, err := GetNewRelicLicenseKey(ctx, conf)
	assert.NoError(t, err)
	assert.Equal(t, "env_fallback_value", lk)
}

func TestNilClientHandling(t *testing.T) {
	originalCfg := cfg
	originalSecretsAPI := secretsAPI
	originalSSMAPI := ssmAPI

	cfg = aws.Config{}
	secretsAPI = nil
	ssmAPI = nil

	defer func() {
		cfg = originalCfg
		secretsAPI = originalSecretsAPI
		ssmAPI = originalSSMAPI
	}()

	t.Run("GetNewRelicLicenseKey with nil clients", func(t *testing.T) {
		ctx := context.Background()

		originalEnv := os.Getenv("NEW_RELIC_LICENSE_KEY")
		os.Unsetenv("NEW_RELIC_LICENSE_KEY")
		defer func() {
			if originalEnv != "" {
				os.Setenv("NEW_RELIC_LICENSE_KEY", originalEnv)
			}
		}()

		conf := &config.Configuration{}
		licenseKey, err := GetNewRelicLicenseKey(ctx, conf)

		assert.Equal(t, "", licenseKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "No license key configured")
	})

	t.Run("GetNewRelicLicenseKey falls back to env var when clients nil", func(t *testing.T) {
		ctx := context.Background()

		expectedKey := "test-license-key-from-env"
		originalEnv := os.Getenv("NEW_RELIC_LICENSE_KEY")
		os.Setenv("NEW_RELIC_LICENSE_KEY", expectedKey)
		defer func() {
			if originalEnv != "" {
				os.Setenv("NEW_RELIC_LICENSE_KEY", originalEnv)
			} else {
				os.Unsetenv("NEW_RELIC_LICENSE_KEY")
			}
		}()

		conf := &config.Configuration{}
		licenseKey, err := GetNewRelicLicenseKey(ctx, conf)

		assert.Equal(t, expectedKey, licenseKey)
		assert.NoError(t, err)
	})

	t.Run("IsSecretConfigured with nil client", func(t *testing.T) {
		ctx := context.Background()
		conf := &config.Configuration{LicenseKeySecretId: "test-secret"}
		assert.False(t, IsSecretConfigured(ctx, conf))
	})

	t.Run("IsSSMParameterConfigured with nil client", func(t *testing.T) {
		ctx := context.Background()
		conf := &config.Configuration{LicenseKeySSMParameterName: "test-param"}

		assert.False(t, IsSSMParameterConfigured(ctx, conf))
	})
}

func TestInitErrorPath(t *testing.T) {
	originalCfg := cfg
	originalSecretsAPI := secretsAPI
	originalSSMAPI := ssmAPI

	secretsAPI = nil
	ssmAPI = nil

	defer func() {
		cfg = originalCfg
		secretsAPI = originalSecretsAPI
		ssmAPI = originalSSMAPI
	}()

	ctx := context.Background()

	t.Run("AWS config load failure - fallback to env var", func(t *testing.T) {
		expectedKey := "fallback-env-license-key"
		originalEnv := os.Getenv("NEW_RELIC_LICENSE_KEY")
		os.Setenv("NEW_RELIC_LICENSE_KEY", expectedKey)
		defer func() {
			if originalEnv != "" {
				os.Setenv("NEW_RELIC_LICENSE_KEY", originalEnv)
			} else {
				os.Unsetenv("NEW_RELIC_LICENSE_KEY")
			}
		}()

		conf := &config.Configuration{}
		licenseKey, err := GetNewRelicLicenseKey(ctx, conf)

		assert.Equal(t, expectedKey, licenseKey)
		assert.NoError(t, err)
	})

	t.Run("AWS config load failure - no fallback available", func(t *testing.T) {
		// Clear environment variable
		originalEnv := os.Getenv("NEW_RELIC_LICENSE_KEY")
		os.Unsetenv("NEW_RELIC_LICENSE_KEY")
		defer func() {
			if originalEnv != "" {
				os.Setenv("NEW_RELIC_LICENSE_KEY", originalEnv)
			}
		}()

		conf := &config.Configuration{}
		licenseKey, err := GetNewRelicLicenseKey(ctx, conf)

		assert.Equal(t, "", licenseKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "No license key configured")
	})

	t.Run("AWS config load failure - secret operations fail gracefully", func(t *testing.T) {
		conf := &config.Configuration{LicenseKeySecretId: "some-secret"}

		assert.False(t, IsSecretConfigured(ctx, conf))

		licenseKey, err := GetNewRelicLicenseKey(ctx, conf)
		assert.Equal(t, "", licenseKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Secrets Manager client not initialized")
	})

	t.Run("AWS config load failure - SSM operations fail gracefully", func(t *testing.T) {
		conf := &config.Configuration{LicenseKeySSMParameterName: "some-parameter"}
		assert.False(t, IsSSMParameterConfigured(ctx, conf))
		licenseKey, err := GetNewRelicLicenseKey(ctx, conf)
		assert.Equal(t, "", licenseKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SSM client not initialized")
	})
}
