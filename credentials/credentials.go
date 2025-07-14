package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/util"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type licenseKeySecret struct {
	LicenseKey string
}

// SecretsManagerAPI defines the interface for Secrets Manager operations
type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// SSMAPI defines the interface for SSM operations
type SSMAPI interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

var (
	cfg        aws.Config
	secretsAPI SecretsManagerAPI
	ssmAPI     SSMAPI
)

const defaultSecretId = "NEW_RELIC_LICENSE_KEY"

func init() {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var err error
	cfg, err = awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		util.Logf("Failed to load AWS config: %v", err)
		return
	}
	secretsAPI = secretsmanager.NewFromConfig(cfg)
	ssmAPI = ssm.NewFromConfig(cfg)
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
	if secretsAPI == nil {
		return false
	}

	secretId := getLicenseKeySecretId(conf)
	secretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}

	_, err := secretsAPI.GetSecretValue(ctx, &secretValueInput)
	if err != nil {
		return false
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
	if secretsAPI == nil {
		return "", fmt.Errorf("Secrets Manager client not initialized")
	}

	util.Debugf("fetching '%s' from Secrets Manager\n", secretId)

	secretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}

	secretValueOutput, err := secretsAPI.GetSecretValue(ctx, &secretValueInput)
	if err != nil {
		return "", err
	}

	return decodeLicenseKey(secretValueOutput.SecretString)
}

func tryLicenseKeyFromSSMParameter(ctx context.Context, parameterName string) (string, error) {
	if ssmAPI == nil {
		return "", fmt.Errorf("SSM client not initialized")
	}

	util.Debugf("fetching '%s' from SSM Parameter Store\n", parameterName)

	parameterValueInput := ssm.GetParameterInput{Name: &parameterName, WithDecryption: aws.Bool(true)}

	parameterValueOutput, err := ssmAPI.GetParameter(ctx, &parameterValueInput)
	if err != nil {
		return "", err
	}

	return *parameterValueOutput.Parameter.Value, nil
}

// OverrideSecretsManager overrides the default Secrets Manager implementation
func OverrideSecretsManager(override SecretsManagerAPI) {
	secretsAPI = override
}

// OverrideSSM overrides the default SSM implementation
func OverrideSSM(override SSMAPI) {
	ssmAPI = override
}
