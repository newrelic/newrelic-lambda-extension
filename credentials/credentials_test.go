package credentials

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"os"
	"testing"
)

func TestGetLicenseKeySecretId(t *testing.T) {
	defaultSecretId := getLicenseKeySecretId()
	if defaultSecretId != defaultSecretId {
		t.Error("Unexpected default value")
	}

	const testSecretId = "testSecretName"
	os.Setenv(secretNameEnvVar, testSecretId)
	customSecretId := getLicenseKeySecretId()
	if customSecretId != testSecretId {
		t.Error("Unexpected custom value")
	}
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

	if *lk != "foo" {
		t.Error("Wrong license key string")
	}
}
