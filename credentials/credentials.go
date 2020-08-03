package credentials

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"os"
)

type LicenseKeySecret struct {
	LicenseKey string
}

var sess = session.Must(session.NewSessionWithOptions(session.Options{
	SharedConfigState: session.SharedConfigEnable,
}))

func getLicenseKeySecretId() string {
	secretId, found := os.LookupEnv("NEW_RELIC_LICENSE_KEY_SECRET_ID")
	if !found {
		return "NEW_RELIC_LICENSE_KEY"
	}
	return secretId
}

func decodeLicenseKey(rawJson *string) (*string, error) {
	secrets := LicenseKeySecret{}
	err := json.Unmarshal([]byte(*rawJson), &secrets)

	if err != nil {
		return nil, err
	}

	return &secrets.LicenseKey, nil
}

func GetNewRelicLicenseKey() (*string, error) {
	secrets := secretsmanager.New(sess)
	secretId := getLicenseKeySecretId()
	secretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}

	secretValueOutput, err := secrets.GetSecretValue(&secretValueInput)
	if err != nil {
		return nil, err
	}
	return decodeLicenseKey(secretValueOutput.SecretString)
}
