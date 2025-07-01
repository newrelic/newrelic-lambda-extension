package checks

import (
	"context"
	"net/http"
	"path/filepath"
	"regexp"
	"time"

	"github.com/google/go-github/v68/github"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type httpClient interface {
	Get(string) (*http.Response, error)
}

var (
	client       httpClient
	githubClient *github.Client
	re           = regexp.MustCompile(`\/releases\/tag\/(v[0-9.]+)`)
)

func init() {
	client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: time.Second * 2,
	}
	githubClient = github.NewClient(&http.Client{Timeout: time.Second * 2})
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
	ctx := context.Background()
	release, _, err := githubClient.Repositories.GetLatestRelease(ctx, r.agentVersionGitOrg, r.agentVersionGitRepo)

	if err != nil {
		util.Debugf("Could not retrieve latest GitHub release: %v", err)
		return nil
	}

	if release.TagName != nil {
		r.AgentVersion = *release.TagName
	}

	return nil
}
