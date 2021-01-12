package checks

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type checkFn func(*config.Configuration, *api.RegistrationResponse) error
type checkRuntime func(handlerConfigs) error

type handlerConfigs struct {
	handlerName string
	conf        *config.Configuration
}

type LogSender interface {
	SendFunctionLogs(lines []logserver.LogLine) error
}

// TODO move this
var NEW_RELIC_LAMBDA_WRAPPER = "newrelic_lambda_wrapper.handler"

/// Register checks here
var checks = []checkFn{
	exampleCheckFunction,
	checkHandler,
}

var runtimeChecks = map[string]checkRuntime{
	"dotnet": dotnet,
	"java":   java,
	"node":   node,
	"python": python,
}

func RunChecks(conf *config.Configuration, reg *api.RegistrationResponse, logSender LogSender) {
	for _, check := range checks {
		_ = runCheck(conf, reg, logSender, check)
	}
}

func runCheck(conf *config.Configuration, reg *api.RegistrationResponse, logSender LogSender, check checkFn) error {
	err := check(conf, reg)
	if err != nil {
		errLog := fmt.Sprintf("Startup check failed: %v", err)
		util.Logln(errLog)

		//Send a log line to NR as well
		_ = logSender.SendFunctionLogs([]logserver.LogLine{
			{
				Time:      time.Now(),
				RequestID: "0",
				Content:   []byte(errLog),
			},
		})
	}
	return err
}

func exampleCheckFunction(*config.Configuration, *api.RegistrationResponse) error {
	return nil
}

// TODO:
// validate that java handler ends with interface method handleRequest
// check on custom runtimes
// tests
func checkHandler(conf *config.Configuration, reg *api.RegistrationResponse) error {
	functionHandler := reg.Handler

	for k, v := range runtimeChecks {

		path := fmt.Sprintf("/var/lang/bin/%s", k)

		_, err := os.Stat(path)

		if err == nil {
			h := handlerConfigs{
				handlerName: functionHandler,
				conf:        conf,
			}
			err = v(h)
			if err != nil {
				return fmt.Errorf("Lambda handler is set incorrectly: %s", err)
			}
			break
		}
	}
	return nil
}

// Handler format for dotnet functions: Assembly::Namespace.ClassName::Method
// from the file names alone all that we can validate is the Assembly dll
func dotnet(h handlerConfigs) error {
	s := strings.Split(h.handlerName, "::")
	_, err := os.Stat("/var/task/" + s[0] + ".dll")
	return err
}

// Handler format for java functions: package.Class::method
// we can validate that the package + class path exist
// and want to validate that the method value is set to ::handleRequest
func java(h handlerConfigs) error {
	functionHandler := strings.ReplaceAll(h.handlerName, ".", "/")
	s := strings.Split(functionHandler, "::")
	_, err := os.Stat("/var/task/" + s[0] + ".class")
	return err
}

// Hander format for node functions: file.function
// we can validate that the handler has been set to newrelic_lambda_wrapper.handler
// and that the file name provided in the env var NEW_RELIC_LAMBDA_HANDLER exists
func node(h handlerConfigs) error {
	functionHandler, err := checkNrHandler(h)
	if err != nil {
		return err
	}
	s := strings.Split(functionHandler, ".")
	_, err = os.Stat("/var/task/" + strings.Join(s[:len(s)-1], "/") + ".js")
	return err
}

// Hander format for python functions: file.function
// we can validate that the handler has been set to newrelic_lambda_wrapper.handler
// and that the file name provided in the env var NEW_RELIC_LAMBDA_HANDLER exists
func python(h handlerConfigs) error {
	functionHandler, err := checkNrHandler(h)
	if err != nil {
		return err
	}
	s := strings.Split(functionHandler, ".")
	_, err = os.Stat("/var/task/" + strings.Join(s[:len(s)-1], "/") + ".py")
	return err
}

func checkNrHandler(h handlerConfigs) (string, error) {
	if h.handlerName != NEW_RELIC_LAMBDA_WRAPPER {
		return "", fmt.Errorf("handler value invalid. Must be set as: %s", NEW_RELIC_LAMBDA_WRAPPER)
	}
	functionHandler := *h.conf.NRHandler
	return functionHandler, nil
}
