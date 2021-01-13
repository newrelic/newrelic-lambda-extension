package checks

import (
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/stretchr/testify/assert"
)

func TestPathFormatterJava(t *testing.T) {
	testHandler := "com.newrelic.lambda.example.App::handleRequest"
	e := "/var/task/com/newrelic/lambda/example/App.java"
	r := pathFormatter(testHandler, "java")
	assert.Equal(t, e, r)
}

func TestJavaMethodError(t *testing.T) {
	h := handlerConfigs{
		handlerName: "com.newrelic.lambda.example.App::foobar",
		conf:        &config.Configuration{},
	}
	err := java(h)
	assert.Error(t, err)
}

func TestPathFormatterDotnet(t *testing.T) {
	testHandler := "NewRelicExampleDotnet::NewRelicExampleDotnet.Function::FunctionHandler"
	e := "/var/task/NewRelicExampleDotnet.dll"
	r := pathFormatter(testHandler, "dll")
	assert.Equal(t, e, r)
}

func TestPathFormatterPython(t *testing.T) {
	testHandler := "path/to/app.lambda_handler"
	e := "/var/task/path/to/app.py"
	r := pathFormatter(testHandler, "py")
	assert.Equal(t, e, r)
}

func TestPathFormatterNode(t *testing.T) {
	testHandler := "path/to/app.lambda_handler"
	e := "/var/task/path/to/app.js"
	r := pathFormatter(testHandler, "js")
	assert.Equal(t, e, r)
}

func TestCheckNrHandlerTrue(t *testing.T) {
	e := "path/to/app.lambda_handler"
	c := config.Configuration{}
	c.NRHandler = &e
	h := handlerConfigs{
		handlerName: "newrelic-lambda-wrapper.handler",
		conf:        &c,
	}
	r := checkNrHandler(h, "-")
	assert.Equal(t, e, r)
}

func TestCheckNrHandlerFalse(t *testing.T) {
	h := handlerConfigs{
		handlerName: "hello-world/app.haldoer",
		conf:        &config.Configuration{},
	}
	r := checkNrHandler(h, "-")
	assert.Equal(t, h.handlerName, r)
}
