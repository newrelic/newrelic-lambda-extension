package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type handlerConfigs struct {
	handlerName string
	conf        *config.Configuration
}

var handlerPath = "/var/task"

func handlerCheck(ctx context.Context, conf *config.Configuration, reg *api.RegistrationResponse, r runtimeConfig) error {
	if r.language != "" {
		h := handlerConfigs{
			handlerName: reg.Handler,
			conf:        conf,
		}

		if !r.check(h) {
			return fmt.Errorf("Missing handler file %s (NEW_RELIC_LAMBDA_HANDLER=%s)", h.handlerName, conf.NRHandler)
		}
	}

	return nil
}

func (r runtimeConfig) check(h handlerConfigs) bool {
	functionHandler := r.getTrueHandler(h)
	p := removePathMethodName(functionHandler)
	p = pathFormatter(p, r.fileType)
	return util.PathExists(p)
}

func (r runtimeConfig) getTrueHandler(h handlerConfigs) string {
	if h.handlerName != r.wrapperName {
		util.Logln("Warning: handler not set to New Relic layer wrapper", r.wrapperName)
		return h.handlerName
	}

	return h.conf.NRHandler
}

func removePathMethodName(p string) string {
	s := strings.Split(p, ".")
	return strings.Join(s[:len(s)-1], "/")
}

func pathFormatter(functionHandler string, fileType string) string {
	p := fmt.Sprintf("%s/%s.%s", handlerPath, functionHandler, fileType)
	return p
}
