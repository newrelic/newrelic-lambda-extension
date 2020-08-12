package credentials

import (
	"testing"

	"github.com/newrelic/lambda-extension/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/stretchr/testify/assert"
)

func TestGetLicenseKeySecretId(t *testing.T) {
	secretId := getLicenseKeySecretId(&config.Configuration{})
	assert.Equal(t, defaultSecretId, secretId)

	var testSecretId = "testSecretName"
	var conf = &config.Configuration{LicenseKeySecretId: &testSecretId}
	secretId = getLicenseKeySecretId(conf)
	assert.Equal(t, testSecretId, secretId)
}

type mockSecretManager struct {
	secretsmanageriface.SecretsManagerAPI
}

func (mockSecretManager) GetSecretValue(*secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(`{"LicenseKey": "foo"}`),
	}, nil
}

func TestGetLicenseKeyImpl(t *testing.T) {
	lk, err := getLicenseKeyImpl(mockSecretManager{}, &config.Configuration{})
	if err != nil {
		t.Error("Unexpected error", err)
	}

	assert.Equal(t, "foo", *lk)
}

func TestGetNewRelicLicenseKeyConfigValue(t *testing.T) {
	licenseKey := "test_value"
	resultKey, err := GetNewRelicLicenseKey(&config.Configuration{
		LicenseKey: &licenseKey,
	})

	assert.Nil(t, err)
	assert.Equal(t, licenseKey, *resultKey)
}
