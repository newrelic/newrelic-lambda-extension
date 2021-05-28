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

type mockSecretManager struct {
	secretsmanageriface.SecretsManagerAPI
}

func (mockSecretManager) GetSecretValueWithContext(context.Context, *secretsmanager.GetSecretValueInput, ...request.Option) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(`{"LicenseKey": "foo"}`),
	}, nil
}

type mockSecretManagerErr struct {
	secretsmanageriface.SecretsManagerAPI
}

func (mockSecretManagerErr) GetSecretValueWithContext(context.Context, *secretsmanager.GetSecretValueInput, ...request.Option) (*secretsmanager.GetSecretValueOutput, error) {
	return nil, fmt.Errorf("Something went wrong")
}

func TestIsSecretConfigured(t *testing.T) {
	OverrideSecretsManager(mockSecretManager{})
	ctx := context.Background()
	assert.True(t, IsSecretConfigured(ctx, &config.Configuration{}))

	OverrideSecretsManager(mockSecretManagerErr{})
	assert.False(t, IsSecretConfigured(ctx, &config.Configuration{}))
}

func TestGetNewRelicLicenseKey(t *testing.T) {
	OverrideSecretsManager(mockSecretManager{})
	ctx := context.Background()
	lk, err := GetNewRelicLicenseKey(ctx, &config.Configuration{})
	assert.Nil(t, err)
	assert.Equal(t, "foo", lk)

	os.Unsetenv("NEW_RELIC_LICENSE_KEY")
	OverrideSecretsManager(mockSecretManagerErr{})
	lk, err = GetNewRelicLicenseKey(ctx, &config.Configuration{})
	assert.Error(t, err)
	assert.Empty(t, lk)

	os.Setenv("NEW_RELIC_LICENSE_KEY", "foobar")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")
	lk, err = GetNewRelicLicenseKey(ctx, &config.Configuration{})
	assert.Nil(t, err)
	assert.Equal(t, "foobar", lk)
}

func TestGetNewRelicLicenseKeyConfigValue(t *testing.T) {
	licenseKey := "test_value"
	ctx := context.Background()
	resultKey, err := GetNewRelicLicenseKey(ctx, &config.Configuration{
		LicenseKey: licenseKey,
	})

	assert.Nil(t, err)
	assert.Equal(t, licenseKey, resultKey)
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
