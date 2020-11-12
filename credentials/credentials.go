package credentials

import (
	"encoding/json"
	"github.com/newrelic/newrelic-lambda-extension/util"
	"os"

	"github.com/newrelic/newrelic-lambda-extension/config"

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
		util.Logln("Fetching license key from secret id " + *conf.LicenseKeySecretId)
		return *conf.LicenseKeySecretId
	}
	return defaultSecretId
}

func decodeLicenseKey(rawJson *string) (*string, error) {
	var secrets licenseKeySecret

	err := json.Unmarshal([]byte(*rawJson), &secrets)
	if err != nil {
		return nil, err
	}

	return &secrets.LicenseKey, nil
}

func getLicenseKeyImpl(secrets secretsmanageriface.SecretsManagerAPI, conf *config.Configuration) (*string, error) {
	secretId := getLicenseKeySecretId(conf)
	secretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}

	secretValueOutput, err := secrets.GetSecretValue(&secretValueInput)
	if err != nil {
		envLicenseKey, found := os.LookupEnv(defaultSecretId)
		if found {
			return &envLicenseKey, nil
		}

		return nil, err
	}

	return decodeLicenseKey(secretValueOutput.SecretString)
}

// GetNewRelicLicenseKey fetches the license key from AWS Secrets Manager.
func GetNewRelicLicenseKey(conf *config.Configuration) (*string, error) {
	if conf.LicenseKey != nil {
		util.Logln("Using license key from environment variable")
		return conf.LicenseKey, nil
	}

	secrets := secretsmanager.New(sess)
	return getLicenseKeyImpl(secrets, conf)
}
