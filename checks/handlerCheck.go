package checks

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
)

type handlerRuntime func(HandlerConfigs) error

type HandlerConfigs struct {
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
	fmt.Println(functionHandler)
	fmt.Println("hello from check handler")
	for k, v := range handlerCheck {
		err := checkRuntime(k)
		if err == nil {
			h := HandlerConfigs{
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
	path := fmt.Sprintf("/var/lang/bin/%s", runtime)
	return pathExists(path)
}

// Handler format for dotnet functions: Assembly::Namespace.ClassName::Method
// from the file names alone all that we can validate is the Assembly dll
func dotnet(h HandlerConfigs) error {
	path := pathFormatter(h.handlerName, "dll")
	return pathExists(path)
}

// Handler format for java functions: package.Class::method
// we can validate that the package + class path exist
// and want to validate that the method value is set to interface method handleRequest
func java(h HandlerConfigs) error {
	if !strings.HasSuffix(h.handlerName, "handleRequest") {
		return fmt.Errorf("Function handler missing required 'handleRequest' method")
	}
	path := pathFormatter(h.handlerName, "class")
	return pathExists(path)
}

// Hander format for node functions: file.function
// we can validate that the handler has been set to newrelic-lambda-wrapper.handler"
// and that the file name provided in the env var NEW_RELIC_LAMBDA_HANDLER exists
func node(h HandlerConfigs) error {
	functionHandler := checkNrHandler(h, "-")
	path := pathFormatter(functionHandler, "js")
	return pathExists(path)
}

// Hander format for python functions: file.function
// we can validate that the handler has been set to newrelic_lambda_wrapper.handler
// and that the file name provided in the env var NEW_RELIC_LAMBDA_HANDLER exists
func python(h HandlerConfigs) error {
	functionHandler := checkNrHandler(h, "_")
	path := pathFormatter(functionHandler, "py")
	return pathExists(path)
}

func checkNrHandler(h HandlerConfigs, dividingChar string) string {
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

	path := fmt.Sprintf("%s/%s.%s", handlerPath, functionHandler, fileType)
	fmt.Println("my path")
	fmt.Println(path)
	return path
}

func pathExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}
	return nil
}
