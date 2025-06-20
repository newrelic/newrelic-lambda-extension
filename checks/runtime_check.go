package checks

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v68/github"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type httpClient interface {
	Get(string) (*http.Response, error)
}

var (
	client        httpClient
	githubClient  *github.Client
	re            = regexp.MustCompile(`\/releases\/tag\/(v[0-9.]+)`)
	osStatFunc    = os.Stat
	osReadDirFunc = os.ReadDir
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

func DetectRuntime() string {
	return detectRuntimeWithFileSystem(osStatFunc, osReadDirFunc)
}

func detectRuntimeWithFileSystem(statFunc func(string) (os.FileInfo, error), readDirFunc func(string) ([]os.DirEntry, error)) string {
	if runtime := os.Getenv("AWS_EXECUTION_ENV"); runtime != "" {
		lowerRuntime := strings.ToLower(runtime)
		if strings.Contains(lowerRuntime, "nodejs") {
			return "Node"
		}
	}

	if handler := os.Getenv("_HANDLER"); handler != "" {
		lowerHandler := strings.ToLower(handler)
		if strings.HasSuffix(lowerHandler, ".js") || strings.HasSuffix(lowerHandler, ".mjs") || strings.HasSuffix(lowerHandler, ".cjs") {
			return "Node"
		}

	}

	if runtimeDir := os.Getenv("LAMBDA_RUNTIME_DIR"); runtimeDir != "" {
		lowerRuntimeDir := strings.ToLower(runtimeDir)
		if strings.Contains(lowerRuntimeDir, "nodejs") {
			return "Node"
		}

	}

	runtimeBinaries := map[string]string{
		"/var/lang/bin/node": "Node",
	}

	for path, runtime := range runtimeBinaries {
		if _, err := statFunc(path); err == nil {
			return runtime
		}
	}

	if runtimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API"); runtimeAPI != "" {
		if taskRoot := os.Getenv("LAMBDA_TASK_ROOT"); taskRoot != "" {
			commonFiles := map[string]string{
				taskRoot + "/package.json": "Node",
			}

			for path, runtime := range commonFiles {
				if _, err := statFunc(path); err == nil {
					return runtime
				}
			}
		}
	}

	return "Unknown"
}
