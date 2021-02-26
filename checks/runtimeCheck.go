package checks

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

var re = regexp.MustCompile(`\/releases\/tag\/(v[0-9.]+)`)

func checkAndReturnRuntime() (runtimeConfig, error) {
	for k, v := range runtimeConfigs {
		p := fmt.Sprintf("%s/%s", runtimeLookupPath, k)
		if util.PathExists(p) {
			err := latestAgentTag(&v)
			return v, err
		}
	}
	// If we make it here that means the runtime is not one we
	// currently validate so we don't want to warn against anything
	return runtimeConfig{}, nil
}

func latestAgentTag(r *runtimeConfig) error {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(r.agentVersionUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != 302 {
		// The version check HTTP request failed; this doesn't tell us anything
		util.Debugf("Can't query latest agent version. Request to %v returned status %v", r.agentVersionUrl, resp.StatusCode)
		return nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	rs := re.FindStringSubmatch(string(body))
	if len(rs) != 2 {
		return errors.New("Can't determine latest agent version.")
	}
	r.AgentVersion = rs[1]
	return nil
}
