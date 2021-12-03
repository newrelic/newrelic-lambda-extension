package checks

import (
	"errors"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

type httpClient interface {
	Get(string) (*http.Response, error)
}

var (
	client httpClient
	re     = regexp.MustCompile(`\/releases\/tag\/(v[0-9.]+)`)
)

func init() {
	client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: time.Second * 10,
	}
}

func checkAndReturnRuntime() (runtimeConfig, error) {
	for k, v := range runtimeConfigs {
		p := filepath.Join(runtimeLookupPath, string(k))
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
	resp, err := client.Get(r.agentVersionUrl)
	if err != nil {
		// Likely a connectivity issue, log the error but skip check
		util.Debugf("Can't query latest agent version. Request to %v returned error %v", r.agentVersionUrl, err)
		return nil
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
