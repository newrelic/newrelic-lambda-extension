package credentials

import (
	"encoding/json"

	"github.com/newrelic/lambda-extension/config"

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

func getLicenseKeySecretId(conf *config.Configuration) string {
	if conf.LicenseKeySecretId != nil {
		return *conf.LicenseKeySecretId
	}
	return defaultSecretId
}

func decodeLicenseKey(rawJson *string) (*string, error) {
	secrets := licenseKeySecret{}
	err := json.Unmarshal([]byte(*rawJson), &secrets)

	if err != nil {
		return nil, err
	}

	return &secrets.LicenseKey, nil
}

func getLicencesKeyImpl(secrets secretsmanageriface.SecretsManagerAPI, conf *config.Configuration) (*string, error) {
	secretId := getLicenseKeySecretId(conf)
	secretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}

	secretValueOutput, err := secrets.GetSecretValue(&secretValueInput)
	if err != nil {
		return nil, err
	}
	return decodeLicenseKey(secretValueOutput.SecretString)
}

// GetNewRelicLicenseKey fetches the license key from AWS Secrets Manager.
func GetNewRelicLicenseKey(conf *config.Configuration) (*string, error) {
	if conf.LicenseKey != nil {
		return conf.LicenseKey, nil
	}

	secrets := secretsmanager.New(sess)
	return getLicencesKeyImpl(secrets, conf)
}
