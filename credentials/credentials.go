package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/newrelic/newrelic-lambda-extension/util"

	"github.com/newrelic/newrelic-lambda-extension/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

type licenseKeySecret struct {
	LicenseKey string
}

var (
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	secrets   secretsmanageriface.SecretsManagerAPI
	ssmClient ssmiface.SSMAPI
)

const defaultSecretId = "NEW_RELIC_LICENSE_KEY"

func init() {
	secrets = secretsmanager.New(sess)
	ssmClient = ssm.New(sess)
}

func getLicenseKeySecretId(conf *config.Configuration) string {
	if conf.LicenseKeySecretId != "" {
		return conf.LicenseKeySecretId
	}

	return defaultSecretId
}

func getLicenseKeySSMParameterName(conf *config.Configuration) string {
	if conf.LicenseKeySSMParameterName != "" {
		return conf.LicenseKeySSMParameterName
	}

	return defaultSecretId
}

func decodeLicenseKey(rawJson *string) (string, error) {
	var secrets licenseKeySecret

	err := json.Unmarshal([]byte(*rawJson), &secrets)
	if err != nil {
		return "", err
	}
	if secrets.LicenseKey == "" {
		return "", fmt.Errorf("malformed license key secret; missing \"LicenseKey\" attribute")
	}

	return secrets.LicenseKey, nil
}

// IsSecretConfigured returns true if the Secrets Manager secret is configured, false
// otherwise
func IsSecretConfigured(ctx context.Context, conf *config.Configuration) bool {
	secretId := getLicenseKeySecretId(conf)
	secretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}
	svc := secretsmanager.New(sess, aws.NewConfig().WithRegion("us-east-1"))
	_, err := secrets.GetSecretValueWithContext(ctx, &secretValueInput)
	if err != nil {
		util.Debugf("Secret '%s' not found in Secrets Manager trying different region\n", secretId)
		input := &secretsmanager.GetSecretValueInput{
			SecretId: &secretId,
		}
	
		_, err := svc.GetSecretValueWithContext(ctx, input)
		if err != nil {
			return false
		}
	}

	return true
}

// IsSSMParameterConfigured returns true if the SSM parameter is configured, false
// otherwise.
func IsSSMParameterConfigured(ctx context.Context, conf *config.Configuration) bool {
	parameterName := getLicenseKeySSMParameterName(conf)

	_, err := tryLicenseKeyFromSSMParameter(ctx, parameterName)
	if err != nil {
		return false
	}

	return true
}

// GetNewRelicLicenseKey fetches the license key from AWS Secrets Manager or
// SSM Parameter Store, falling back to the NEW_RELIC_LICENSE_KEY environment
// variable if set.
func GetNewRelicLicenseKey(ctx context.Context, conf *config.Configuration) (string, error) {
	if conf.LicenseKey != "" {
		util.Logln("Using license key from environment variable")
		return conf.LicenseKey, nil
	}

	secretId := conf.LicenseKeySecretId
	if secretId != "" {
		util.Logln("Fetching license key from secret id " + secretId)
		return tryLicenseKeyFromSecret(ctx, secretId)
	}

	parameterName := conf.LicenseKeySSMParameterName
	if parameterName != "" {
		util.Logln("Fetching license key from parameter name " + conf.LicenseKeySSMParameterName)
		return tryLicenseKeyFromSSMParameter(ctx, parameterName)
	}

	envLicenseKey, found := os.LookupEnv(defaultSecretId)
	if found {
		return envLicenseKey, nil
	}

	util.Debugln("No configured license key found, attempting fallbacks to default")

	licenseKey, err := tryLicenseKeyFromSecret(ctx, defaultSecretId)
	if err == nil {
		return licenseKey, nil
	}

	licenseKey, err = tryLicenseKeyFromSSMParameter(ctx, defaultSecretId)
	if err == nil {
		return licenseKey, nil
	}

	return "", fmt.Errorf("No license key configured")
}

func tryLicenseKeyFromSecret(ctx context.Context, secretId string) (string, error) {
	util.Debugf("fetching '%s' from Secrets Manager\n", secretId)

	secretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}

	secretValueOutput, err := secrets.GetSecretValueWithContext(ctx, &secretValueInput)
	if err != nil {
		util.Debugf("Secret '%s' not found in Secrets Manager trying different region\n", secretId)
		svc := secretsmanager.New(sess, aws.NewConfig().WithRegion("us-east-1"))
		secretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}
		secretValueOutput, err := svc.GetSecretValueWithContext(ctx, &secretValueInput)
		util.Debugf("secretValueOutput: %v\n", secretValueOutput)
		if err != nil {
			return "", err
		}
		return decodeLicenseKey(secretValueOutput.SecretString)
	}

	return decodeLicenseKey(secretValueOutput.SecretString)
}

func tryLicenseKeyFromSSMParameter(ctx context.Context, parameterName string) (string, error) {
	util.Debugf("fetching '%s' from SSM Parameter Store\n", parameterName)

	parameterValueInput := ssm.GetParameterInput{Name: &parameterName, WithDecryption: aws.Bool(true)}

	parameterValueOutput, err := ssmClient.GetParameterWithContext(ctx, &parameterValueInput)
	if err != nil {
		return "", err
	}

	return *parameterValueOutput.Parameter.Value, nil
}

// OverrideSecretsManager overrides the default Secrets Manager implementation
func OverrideSecretsManager(override secretsmanageriface.SecretsManagerAPI) {
	secrets = override
}

// OverrideSSM overrides the default SSM implementation
func OverrideSSM(override ssmiface.SSMAPI) {
	ssmClient = override
}
