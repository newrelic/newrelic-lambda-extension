package checks

import (
	"fmt"
	"os"
	"regexp"
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
	"dotnet": dotnet,
	"java":   java,
	"node":   node,
	"python": python,
}

var NEW_RELIC_LAMBDA_WRAPPER = "newrelic|lambda|wrapper.handler"
var handlerPath = "/var/task"

func checkHandler(conf *config.Configuration, reg *api.RegistrationResponse) error {
	functionHandler := reg.Handler
	for k, v := range handlerCheck {
		err := checkRuntime(k)
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
	// If we make it here that means the runtime is
	// custom and we don't want to throw an error
	return nil
}

func checkRuntime(runtime string) error {
	p := fmt.Sprintf("/var/lang/bin/%s", runtime)
	return pathExists(p)
}

// Handler format for dotnet functions: Assembly::Namespace.ClassName::Method
// from the file names alone all that we can validate is the Assembly dll
func dotnet(h handlerConfigs) error {
	p := pathFormatter(h.handlerName, "dll")
	return pathExists(p)
}

// Handler format for java functions: package.Class::method
// we can validate that the package + class path exist
// and want to validate that the method value is set to interface method handleRequest
func java(h handlerConfigs) error {
	if !strings.HasSuffix(h.handlerName, "handleRequest") {
		return fmt.Errorf("Function handler missing required 'handleRequest' method")
	}
	p := pathFormatter(h.handlerName, "class")
	return pathExists(p)
}

// Hander format for node functions: file.function
// we can validate that the handler has been set to newrelic-lambda-wrapper.handler"
// and that the file name provided in the env var NEW_RELIC_LAMBDA_HANDLER exists
func node(h handlerConfigs) error {
	functionHandler := checkNrHandler(h, "-")
	p := pathFormatter(functionHandler, "js")
	return pathExists(p)
}

// Hander format for python functions: file.function
// we can validate that the handler has been set to newrelic_lambda_wrapper.handler
// and that the file name provided in the env var NEW_RELIC_LAMBDA_HANDLER exists
func python(h handlerConfigs) error {
	functionHandler := checkNrHandler(h, "_")
	p := pathFormatter(functionHandler, "py")
	return pathExists(p)
}

func checkNrHandler(h handlerConfigs, dividingChar string) string {
	wrapper := strings.ReplaceAll(NEW_RELIC_LAMBDA_WRAPPER, "|", dividingChar)
	if h.handlerName != wrapper {
		return h.handlerName
	}
	return *h.conf.NRHandler
}

func pathFormatter(functionHandler string, fileType string) string {
	re := regexp.MustCompile("[::.]+")
	s := re.Split(functionHandler, -1)

	if fileType == "dll" {
		functionHandler = s[0]
	} else {
		functionHandler = strings.Join(s[:len(s)-1], "/")
	}

	p := fmt.Sprintf("%s/%s.%s", handlerPath, functionHandler, fileType)
	return p
}

func pathExists(p string) error {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return err
	}
	return nil
}
