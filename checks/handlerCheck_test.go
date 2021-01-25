package checks

import (
	"os"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
)

var testHandler = "path/to/app.handler"

func TestRuntimeMethods(t *testing.T) {
	conf := config.Configuration{}
	r := runtimeConfigs[Node]
	h := handlerConfigs{
		handlerName: r.wrapperName,
		conf:        &conf,
	}
	conf.NRHandler = &testHandler

	t1 := r.getTrueHandler(h)
	t2 := removePathMethodName(t1)
	t3 := pathFormatter(t2, r.fileType)

	e1 := testHandler
	e2 := "path/to/app"
	e3 := "/var/task/path/to/app.js"

	assert.Equal(t, e1, t1)
	assert.Equal(t, e2, t2)
	assert.Equal(t, e3, t3)

	r = runtimeConfigs[Python]

	h = handlerConfigs{
		handlerName: r.wrapperName,
		conf:        &conf,
	}

	t1 = r.getTrueHandler(h)
	t2 = removePathMethodName(t1)
	t3 = pathFormatter(t2, r.fileType)

	e1 = testHandler
	e2 = "path/to/app"
	e3 = "/var/task/path/to/app.py"

	assert.Equal(t, e1, t1)
	assert.Equal(t, e2, t2)
	assert.Equal(t, e3, t3)
}

func TestHandlerCheck(t *testing.T) {
	conf := config.Configuration{}
	reg := api.RegistrationResponse{}
	r := runtimeConfigs[Node]

	// No Runtime
	err := checkHandler(&conf, &reg, runtimeConfig{})
	assert.Nil(t, err)

	// Error
	reg.Handler = testHandler
	conf.NRHandler = &config.EmptyNRWrapper
	err = checkHandler(&conf, &reg, r)
	assert.EqualError(t, err, "Missing handler file path/to/app.handler (NEW_RELIC_LAMBDA_HANDLER=Undefined)")

	// Success
	dirname, err := os.Getwd()

	// Want to make sure our working directory isn't root
	assert.NotEqual(t, dirname, "")
	assert.Nil(t, err)

	handlerPath = dirname + "/var/task"
	os.MkdirAll(dirname+"/var/task/path/to/", os.ModePerm)
	os.Create(dirname + "/var/task/path/to/app.js")
	defer os.RemoveAll(dirname + "/var")
	reg.Handler = testHandler
	conf.NRHandler = &config.EmptyNRWrapper
	err = checkHandler(&conf, &reg, r)
	assert.Nil(t, err)
}
