package credentials

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/stretchr/testify/assert"
)

func TestGetLicenseKeySecretId(t *testing.T) {
	secretId := getLicenseKeySecretId()
	assert.Equal(t, defaultSecretId, secretId)

	const testSecretId = "testSecretName"
	os.Setenv(secretNameEnvVar, testSecretId)
	defer os.Unsetenv(secretNameEnvVar)
	secretId = getLicenseKeySecretId()
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
	lk, err := getLicencesKeyImpl(mockSecretManager{})
	if err != nil {
		t.Error("Unexpected error", err)
	}

	assert.Equal(t, "foo", *lk)
}
