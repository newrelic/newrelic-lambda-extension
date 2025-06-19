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

func DetectRuntime() string {
	if runtime := os.Getenv("AWS_EXECUTION_ENV"); runtime != "" {
		lowerRuntime := strings.ToLower(runtime)
		if strings.Contains(lowerRuntime, "nodejs") {
			return "Node"
		}
		if strings.Contains(lowerRuntime, "python") {
			return "Python"
		}
		if strings.Contains(lowerRuntime, "ruby") {
			return "Ruby"
		}
		if strings.Contains(lowerRuntime, "java") {
			return "Java"
		}
		if strings.Contains(lowerRuntime, "dotnet") {
			return "Dotnet"
		}
		if strings.Contains(lowerRuntime, "go") {
			return "Go"
		}
	}

	if handler := os.Getenv("_HANDLER"); handler != "" {
		lowerHandler := strings.ToLower(handler)
		if strings.HasSuffix(lowerHandler, ".js") || strings.HasSuffix(lowerHandler, ".mjs") || strings.HasSuffix(lowerHandler, ".cjs") {
			return "Node"
		}
		if strings.HasSuffix(lowerHandler, ".py") {
			return "Python"
		}
		if strings.HasSuffix(lowerHandler, ".rb") {
			return "Ruby"
		}
		if strings.Contains(lowerHandler, ".jar") || strings.Contains(lowerHandler, "::") {
			return "Java"
		}
		if strings.HasSuffix(lowerHandler, ".dll") {
			return "Dotnet"
		}
	}

	if runtimeDir := os.Getenv("LAMBDA_RUNTIME_DIR"); runtimeDir != "" {
		lowerRuntimeDir := strings.ToLower(runtimeDir)
		if strings.Contains(lowerRuntimeDir, "nodejs") {
			return "Node"
		}
		if strings.Contains(lowerRuntimeDir, "python") {
			return "Python"
		}
		if strings.Contains(lowerRuntimeDir, "ruby") {
			return "Ruby"
		}
		if strings.Contains(lowerRuntimeDir, "java") {
			return "Java"
		}
		if strings.Contains(lowerRuntimeDir, "dotnet") {
			return "Dotnet"
		}
		if strings.Contains(lowerRuntimeDir, "go") {
			return "Go"
		}
	}

	runtimeBinaries := map[string]string{
		"/var/lang/bin/node":    "Node",
		"/var/lang/bin/python":  "Python",
		"/var/lang/bin/python3": "Python",
		"/var/lang/bin/ruby":    "Ruby",
		"/var/lang/bin/java":    "Java",
		"/usr/bin/dotnet":       "Dotnet",
		"/var/lang/bin/go":      "Go",
	}

	for path, runtime := range runtimeBinaries {
		if _, err := os.Stat(path); err == nil {
			return runtime
		}
	}

	runtimePaths := map[string]string{
		"/var/lang/lib/python3.9":  "Python",
		"/var/lang/lib/python3.10": "Python",
		"/var/lang/lib/python3.11": "Python",
		"/var/lang/lib/python3.12": "Python",
		"/var/lang/lib/python3.13": "Python",
		"/var/lang/lib/ruby":       "Ruby",
		"/var/lang/lib/java":       "Java",
		"/var/runtime":             "Dotnet",
	}

	for path, runtime := range runtimePaths {
		if _, err := os.Stat(path); err == nil {
			return runtime
		}
	}

	if runtimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API"); runtimeAPI != "" {
		if taskRoot := os.Getenv("LAMBDA_TASK_ROOT"); taskRoot != "" {
			commonFiles := map[string]string{
				taskRoot + "/package.json":     "Node",
				taskRoot + "/requirements.txt": "Python",
				taskRoot + "/Gemfile":          "Ruby",
				taskRoot + "/pom.xml":          "Java",
				taskRoot + "/build.gradle":     "Java",
				taskRoot + "/go.mod":           "Go",
			}

			for path, runtime := range commonFiles {
				if _, err := os.Stat(path); err == nil {
					return runtime
				}
			}

			// Check for .csproj files (wildcard pattern)
			if entries, err := os.ReadDir(taskRoot); err == nil {
				for _, entry := range entries {
					if strings.HasSuffix(strings.ToLower(entry.Name()), ".csproj") {
						return "Dotnet"
					}
				}
			}
		}
	}

	return "Unknown"
}
