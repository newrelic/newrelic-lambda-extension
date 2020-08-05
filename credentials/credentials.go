package credentials

import (
	"encoding/json"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
)

type licenseKeySecret struct {
	LicenseKey string
}

var sess = session.Must(session.NewSessionWithOptions(session.Options{
	SharedConfigState: session.SharedConfigEnable,
}))

const defaultSecretId = "NEW_RELIC_LICENSE_KEY"
const secretNameEnvVar = "NEW_RELIC_LICENSE_KEY_SECRET_ID"

func getLicenseKeySecretId() string {
	secretId, found := os.LookupEnv(secretNameEnvVar)
	if !found {
		return defaultSecretId
	}
	return secretId
}

func decodeLicenseKey(rawJson *string) (*string, error) {
	secrets := licenseKeySecret{}
	err := json.Unmarshal([]byte(*rawJson), &secrets)

	if err != nil {
		return nil, err
	}

	return &secrets.LicenseKey, nil
}

func getLicencesKeyImpl(secrets secretsmanageriface.SecretsManagerAPI) (*string, error) {
	secretId := getLicenseKeySecretId()
	secretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}

	secretValueOutput, err := secrets.GetSecretValue(&secretValueInput)
	if err != nil {
		return nil, err
	}
	return decodeLicenseKey(secretValueOutput.SecretString)
}

// GetNewRelicLicenseKey fetches the license key from AWS Secrets Manager.
func GetNewRelicLicenseKey() (*string, error) {
	secrets := secretsmanager.New(sess)
	return getLicencesKeyImpl(secrets)
}
