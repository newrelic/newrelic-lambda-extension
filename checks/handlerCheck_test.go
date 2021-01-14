package checks

import (
	"errors"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
)

var testHandler = "path/to/app.handler"
var conf = config.Configuration{}
var reg = api.RegistrationResponse{}

func TestHandlerCheckSuccess(t *testing.T) {

	pathExists = func(p string) error {
		return nil
	}
	checkAndReturnRuntime = func() RuntimeHandlerCheck {
		return handlerCheck["node"]
	}

	err := checkHandler(&conf, &reg)
	assert.Nil(t, err)
}

func TestHandlerCheckError(t *testing.T) {
	pathExists = func(p string) error {
		return errors.New("Error!")
	}
	checkAndReturnRuntime = func() RuntimeHandlerCheck {
		return handlerCheck["node"]
	}
	reg.Handler = testHandler
	conf.NRHandler = &config.EmptyNRWrapper
	err := checkHandler(&conf, &reg)
	assert.EqualError(t, err, "Missing handler file path/to/app.handler (NEW_RELIC_LAMBDA_HANDLER=Undefined): Error!")
}

func TestNode(t *testing.T) {
	w := wrapperCheck{
		wrapperName: "newrelic-lambda-wrapper.handler",
		fileType:    "js",
	}

	h := handlerConfigs{
		handlerName: w.wrapperName,
		conf:        &conf,
	}

	conf.NRHandler = &testHandler

	t1 := w.getTrueHandler(h)
	t2 := w.removePathMethodName(t1)
	t3 := pathFormatter(t2, w.fileType)

	e1 := testHandler
	e2 := "path/to/app"
	e3 := "/var/task/path/to/app.js"

	assert.Equal(t, t1, e1)
	assert.Equal(t, t2, e2)
	assert.Equal(t, t3, e3)
}

func TestPython(t *testing.T) {
	w := wrapperCheck{
		wrapperName: "newrelic_lambda_wrapper.handler",
		fileType:    "py",
	}

	h := handlerConfigs{
		handlerName: w.wrapperName,
		conf:        &conf,
	}

	conf.NRHandler = &testHandler

	t1 := w.getTrueHandler(h)
	t2 := w.removePathMethodName(t1)
	t3 := pathFormatter(t2, w.fileType)

	e1 := testHandler
	e2 := "path/to/app"
	e3 := "/var/task/path/to/app.py"

	assert.Equal(t, t1, e1)
	assert.Equal(t, t2, e2)
	assert.Equal(t, t3, e3)
}
