package checks

import (
	"errors"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
)

var testHandler = "path/to/app.handler"

func TestHandlerCheckSuccess(t *testing.T) {
	pathExists = func(p string) error {
		return nil
	}
	checkAndReturnRuntime = func() handlerRuntime {
		return node
	}
	conf := config.Configuration{}
	resp := api.RegistrationResponse{}
	err := checkHandler(&conf, &resp)

	assert.Nil(t, err)
}

func TestHandlerCheckError(t *testing.T) {
	pathExists = func(p string) error {
		return errors.New("Error!")
	}
	checkAndReturnRuntime = func() handlerRuntime {
		return node
	}
	conf := config.Configuration{}
	resp := api.RegistrationResponse{}
	err := checkHandler(&conf, &resp)

	assert.EqualError(t, err, "Lambda handler is set incorrectly: Error!")
}

func TestPathPython(t *testing.T) {
	c := config.Configuration{NRHandler: &testHandler}
	h := handlerConfigs{
		handlerName: "newrelic_lambda_wrapper.handler",
		conf:        &c,
	}
	t1 := getTrueHandler(h, "newrelic_lambda_wrapper.handler")
	t2 := removePathMethodName(t1)
	t3 := pathFormatter(t2, "py")

	e1 := "path/to/app.handler"
	e2 := "path/to/app"
	e3 := "/var/task/path/to/app.py"

	assert.Equal(t, e1, e1)
	assert.Equal(t, e2, t2)
	assert.Equal(t, e3, t3)
}

func TestPathNode(t *testing.T) {
	c := config.Configuration{NRHandler: &testHandler}
	h := handlerConfigs{
		handlerName: "newrelic-lambda-wrapper.handler",
		conf:        &c,
	}
	t1 := getTrueHandler(h, "newrelic-lambda-wrapper.handler")
	t2 := removePathMethodName(t1)
	t3 := pathFormatter(t2, "js")

	e1 := "path/to/app.handler"
	e2 := "path/to/app"
	e3 := "/var/task/path/to/app.js"

	assert.Equal(t, e1, e1)
	assert.Equal(t, e2, t2)
	assert.Equal(t, e3, t3)
}

func TestGetTrueHandlerWith(t *testing.T) {
	e := "path/to/app.lambda_handler"
	c := config.Configuration{}
	c.NRHandler = &e
	h := handlerConfigs{
		handlerName: "newrelic-lambda-wrapper.handler",
		conf:        &c,
	}
	r := getTrueHandler(h, "newrelic-lambda-wrapper.handler")
	assert.Equal(t, e, r)
}

func TestGetTrueHandlerWithout(t *testing.T) {
	h := handlerConfigs{
		handlerName: testHandler,
		conf:        &config.Configuration{},
	}
	r := getTrueHandler(h, "newrelic_lambda_wrapper.handler")
	assert.Equal(t, h.handlerName, r)
}
