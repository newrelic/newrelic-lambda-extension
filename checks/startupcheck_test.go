package checks

import (
	"fmt"
	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestLogSender struct {
	sent []logserver.LogLine
}

func (c *TestLogSender) SendFunctionLogs(lines []logserver.LogLine) error {
	c.sent = append(c.sent, lines...)
	return nil
}

func TestRunCheck(t *testing.T) {
	conf := config.Configuration{}
	resp := api.RegistrationResponse{}
	client := TestLogSender{}

	tested := false
	testCheck := func (conf *config.Configuration, resp *api.RegistrationResponse) error {
		tested = true
		return nil
	}

	result := runCheck(&conf, &resp, &client, testCheck)

	assert.Equal(t, true, tested)
	assert.Nil(t, result)
}

func TestRunCheck_err(t *testing.T) {
	conf := config.Configuration{}
	resp := api.RegistrationResponse{}
	logSender := TestLogSender{}

	tested := false
	testCheck := func (conf *config.Configuration, resp *api.RegistrationResponse) error {
		tested = true
		return fmt.Errorf("Failure Test")
	}

	result := runCheck(&conf, &resp, &logSender, testCheck)

	assert.Equal(t, true, tested)
	assert.NotNil(t, result)

	assert.Equal(t, "Startup check failed: Failure Test", string(logSender.sent[0].Content))
}

