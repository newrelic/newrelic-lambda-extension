package checks

import (
	"fmt"
	"os"
	"strings"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type RuntimeHandlerCheck interface {
	check(handlerConfigs) error
}

type wrapperCheck struct {
	wrapperName string
	fileType    string
}

type handlerConfigs struct {
	handlerName string
	conf        *config.Configuration
}

var handlerCheck = map[string]RuntimeHandlerCheck{
	"node": wrapperCheck{
		wrapperName: "newrelic-lambda-wrapper.handler",
		fileType:    "js",
	},
	"python": wrapperCheck{
		wrapperName: "newrelic_lambda_wrapper.handler",
		fileType:    "py",
	},
}

var pathExists = func(p string) error {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return err
	}
	return nil
}

var checkAndReturnRuntime = func() RuntimeHandlerCheck {
	for k, v := range handlerCheck {
		p := fmt.Sprintf("/var/lang/bin/%s", k)
		err := pathExists(p)

		if err == nil {
			return v
		}
	}
	// If we make it here that means the runtime is not one we
	// currently validate so we don't want to warn against anything
	return nil
}

const handlerPath = "/var/task"

func checkHandler(conf *config.Configuration, reg *api.RegistrationResponse) error {
	r := checkAndReturnRuntime()
	if r != nil {
		h := handlerConfigs{
			handlerName: reg.Handler,
			conf:        conf,
		}
		err := r.check(h)
		if err != nil {
			return fmt.Errorf("Missing handler file %s (NEW_RELIC_LAMBDA_HANDLER=%s): %s", h.handlerName, *conf.NRHandler, err)
		}
	}
	return nil
}

func (w wrapperCheck) check(h handlerConfigs) error {
	functionHandler := w.getTrueHandler(h)
	p := w.removePathMethodName(functionHandler)
	p = pathFormatter(p, w.fileType)
	return pathExists(p)
}

func (w wrapperCheck) getTrueHandler(h handlerConfigs) string {
	if h.handlerName != w.wrapperName {
		util.Logln("Warning: handler not set to New Relic layer wrapper", w.wrapperName)
		return h.handlerName
	}
	return *h.conf.NRHandler
}

func (w wrapperCheck) removePathMethodName(p string) string {
	s := strings.Split(p, ".")
	return strings.Join(s[:len(s)-1], "/")
}

func pathFormatter(functionHandler string, fileType string) string {
	p := fmt.Sprintf("%s/%s.%s", handlerPath, functionHandler, fileType)
	return p
}
