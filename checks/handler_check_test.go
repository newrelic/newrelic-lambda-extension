package checks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
)

var testHandler = "path/to/app.handler"

func TestRuntimeMethods(t *testing.T) {
	conf := config.Configuration{TestingOverride: true}
	r := runtimeConfigs[Node]
	h := handlerConfigs{
		handlerName: r.wrapperName,
		conf:        &conf,
	}
	conf.NRHandler = testHandler

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

func TestHandlerCheckJS(t *testing.T) {
	conf := config.Configuration{TestingOverride: true}
	reg := api.RegistrationResponse{}
	r := runtimeConfigs[Node]
	ctx := context.Background()

	// No Runtime
	err := handlerCheck(ctx, &conf, &reg, runtimeConfig{})
	assert.Nil(t, err)

	// Error
	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.EqualError(t, err, "missing handler file path/to/app.handler (NEW_RELIC_LAMBDA_HANDLER=Undefined)")

	// Success
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	handlerPath = filepath.Join(dirname, "var", "task")
	os.MkdirAll(filepath.Join(handlerPath, "path", "to"), os.ModePerm)
	os.Create(filepath.Join(handlerPath, "path", "to", "app.js"))

	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.Nil(t, err)
}

func TestHandlerCheckMJS(t *testing.T) {
	conf := config.Configuration{TestingOverride: true}
	reg := api.RegistrationResponse{}
	r := runtimeConfigs[Node]
	ctx := context.Background()

	// No Runtime
	err := handlerCheck(ctx, &conf, &reg, runtimeConfig{})
	assert.Nil(t, err)

	// Error
	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.EqualError(t, err, "missing handler file path/to/app.handler (NEW_RELIC_LAMBDA_HANDLER=Undefined)")

	// Success
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	handlerPath = filepath.Join(dirname, "var", "task")
	os.MkdirAll(filepath.Join(handlerPath, "path", "to"), os.ModePerm)
	os.Create(filepath.Join(handlerPath, "path", "to", "app.mjs"))

	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.Nil(t, err)
}

func TestHandlerCheckCJS(t *testing.T) {
	conf := config.Configuration{TestingOverride: true}
	reg := api.RegistrationResponse{}
	r := runtimeConfigs[Node]
	ctx := context.Background()

	// No Runtime
	err := handlerCheck(ctx, &conf, &reg, runtimeConfig{})
	assert.Nil(t, err)

	// Error
	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.EqualError(t, err, "missing handler file path/to/app.handler (NEW_RELIC_LAMBDA_HANDLER=Undefined)")

	// Success
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	handlerPath = filepath.Join(dirname, "var", "task")
	os.MkdirAll(filepath.Join(handlerPath, "path", "to"), os.ModePerm)
	os.Create(filepath.Join(handlerPath, "path", "to", "app.cjs"))

	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.Nil(t, err)
}

func TestHandlerCheckPython(t *testing.T) {
	conf := config.Configuration{TestingOverride: true}
	reg := api.RegistrationResponse{}
	r := runtimeConfigs[Python]
	ctx := context.Background()

	// No Runtime
	err := handlerCheck(ctx, &conf, &reg, runtimeConfig{})
	assert.Nil(t, err)

	// Error
	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.EqualError(t, err, "missing handler file path/to/app.handler (NEW_RELIC_LAMBDA_HANDLER=Undefined)")

	// Success
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	handlerPath = filepath.Join(dirname, "var", "task")
	os.MkdirAll(filepath.Join(handlerPath, "path", "to"), os.ModePerm)
	os.Create(filepath.Join(handlerPath, "path", "to", "app.py"))

	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.Nil(t, err)
}
