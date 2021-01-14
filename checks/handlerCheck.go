package checks

import (
	"fmt"
	"os"
	"strings"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
)

type handlerRuntime func(handlerConfigs) error

type handlerConfigs struct {
	handlerName string
	conf        *config.Configuration
}

var handlerCheck = map[string]handlerRuntime{
	"node":   node,
	"python": python,
}

var pathExists = func(p string) error {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return err
	}
	return nil
}

var checkAndReturnRuntime = func() handlerRuntime {
	for k, v := range handlerCheck {
		p := fmt.Sprintf("/var/lang/bin/%s", k)
		err := pathExists(p)
		if err == nil {
			return v
		}
	}
	// If we make it here that means the runtime is
	// custom and we don't want to throw an error
	return nil
}

var handlerPath = "/var/task"

func checkHandler(conf *config.Configuration, reg *api.RegistrationResponse) error {
	r := checkAndReturnRuntime()
	if r != nil {
		h := handlerConfigs{
			handlerName: reg.Handler,
			conf:        conf,
		}
		err := r(h)
		if err != nil {
			return fmt.Errorf("Lambda handler is set incorrectly: %s", err)
		}
	}
	return nil
}

// Handler format for node functions: file.function
// we can validate that the handler has been set to newrelic-lambda-wrapper.handler"
// and that the file name provided in the env var NEW_RELIC_LAMBDA_HANDLER exists
func node(h handlerConfigs) error {
	functionHandler := getTrueHandler(h, "newrelic-lambda-wrapper.handler")
	p := removePathMethodName(functionHandler)
	p = pathFormatter(p, "js")
	return pathExists(p)
}

// Handler format for python functions: file.function
// we can validate that the handler has been set to newrelic_lambda_wrapper.handler
// and that the file name provided in the env var NEW_RELIC_LAMBDA_HANDLER exists
func python(h handlerConfigs) error {
	functionHandler := getTrueHandler(h, "newrelic_lambda_wrapper.handler")
	p := removePathMethodName(functionHandler)
	p = pathFormatter(functionHandler, "py")
	return pathExists(p)
}

func getTrueHandler(h handlerConfigs, w string) string {
	if h.handlerName != w {
		return h.handlerName
	}
	return *h.conf.NRHandler
}

func pathFormatter(functionHandler string, fileType string) string {
	p := fmt.Sprintf("%s/%s.%s", handlerPath, functionHandler, fileType)
	return p
}

func removePathMethodName(p string) string {
	s := strings.Split(p, ".")
	return strings.Join(s[:len(s)-1], "/")
}
