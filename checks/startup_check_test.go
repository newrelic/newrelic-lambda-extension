package checks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/stretchr/testify/assert"
)

type mockClientError struct{}

func (c *mockClientError) Get(string) (*http.Response, error) {
	return nil, errors.New("Something went wrong")
}

type TestLogSender struct {
	sent []logserver.LogLine
}

func (c *TestLogSender) SendFunctionLogs(ctx context.Context, invokedFunctionARN string, lines []logserver.LogLine, entityGuid string) error {
	c.sent = append(c.sent, lines...)
	return nil
}

func TestRunCheck(t *testing.T) {
	conf := config.Configuration{}
	resp := api.RegistrationResponse{}
	r := runtimeConfig{}
	client := TestLogSender{}
	ctx := context.Background()

	tested := false
	testCheck := func(ctx context.Context, conf *config.Configuration, resp *api.RegistrationResponse, r runtimeConfig) error {
		tested = true
		return nil
	}

	result := runCheck(ctx, &conf, &resp, r, &client, testCheck)

	assert.Equal(t, true, tested)
	assert.Nil(t, result)
}

func TestRunCheckErr(t *testing.T) {
	conf := config.Configuration{}
	resp := api.RegistrationResponse{}
	r := runtimeConfig{}
	logSender := TestLogSender{}
	ctx := context.Background()

	tested := false
	testCheck := func(ctx context.Context, conf *config.Configuration, resp *api.RegistrationResponse, r runtimeConfig) error {
		tested = true
		return fmt.Errorf("Failure Test")
	}

	result := runCheck(ctx, &conf, &resp, r, &logSender, testCheck)

	assert.Equal(t, true, tested)
	assert.NotNil(t, result)

	assert.Equal(t, "Startup check warning: Failure Test", string(logSender.sent[0].Content))
}

func TestRunChecks(t *testing.T) {
	c := &config.Configuration{}
	r := &api.RegistrationResponse{}
	l := &TestLogSender{}

	client = &mockClientError{}

	ctx := context.Background()
	RunChecks(ctx, c, r, l)
}

func TestRunChecksIgnoreExtensionChecks(t *testing.T) {
	c := &config.Configuration{IgnoreExtensionChecks: map[string]bool{"agent": true}}
	r := &api.RegistrationResponse{}
	l := &TestLogSender{}

	client = &mockClientError{}

	ctx := context.Background()
	RunChecks(ctx, c, r, l)
}

func TestRunChecksIgnoreExtensionChecksAll(t *testing.T) {
	c := &config.Configuration{IgnoreExtensionChecks: map[string]bool{"all": true}}
	r := &api.RegistrationResponse{}
	l := &TestLogSender{}

	client = &mockClientError{}

	ctx := context.Background()
	RunChecks(ctx, c, r, l)
}
