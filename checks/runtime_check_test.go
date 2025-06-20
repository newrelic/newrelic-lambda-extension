//go:build !race
// +build !race

package checks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntimeCheck(t *testing.T) {
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	oldPath := runtimeLookupPath
	defer func() {
		runtimeLookupPath = oldPath
	}()
	runtimeLookupPath = filepath.Join(dirname, runtimeLookupPath)

	os.MkdirAll(filepath.Join(runtimeLookupPath, "node"), os.ModePerm)
	r, err := checkAndReturnRuntime()
	assert.Equal(t, runtimeConfigs[Node].language, r.language)
	assert.Nil(t, err)
}

func TestRuntimeCheckNil(t *testing.T) {
	r, err := checkAndReturnRuntime()
	assert.Equal(t, runtimeConfig{}, r)
	assert.Nil(t, err)
}

func TestLatestAgentTag(t *testing.T) {
	r := &runtimeConfig{agentVersionGitOrg: runtimeConfigs[Python].agentVersionGitOrg, agentVersionGitRepo: runtimeConfigs[Python].agentVersionGitRepo}
	err := latestAgentTag(r)
	assert.NotEmpty(t, r.AgentVersion)
	assert.Nil(t, err)
}

func TestLatestAgentTagError(t *testing.T) {
	r := &runtimeConfig{agentVersionGitOrg: "", agentVersionGitRepo: ""}
	err := latestAgentTag(r)
	assert.Empty(t, r.AgentVersion)
	assert.Nil(t, err)
}
func mockStatNotFound(path string) (os.FileInfo, error) {
	return nil, errors.New("file not found")
}

func mockStatFound(path string) (os.FileInfo, error) {
	return nil, nil
}

func mockReadDirEmpty(path string) ([]os.DirEntry, error) {
	return []os.DirEntry{}, nil
}

func TestDetectRuntimeFromAWSExecutionEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{"Python Runtime", "AWS_Lambda_python3.13", "Unknown"},
		{"Node Runtime", "AWS_Lambda_nodejs20.x", "Node"},
		{"Java Runtime", "AWS_Lambda_java21", "Unknown"},
		{"Ruby Runtime", "AWS_Lambda_ruby3.2", "Unknown"},
		{"Dotnet Runtime", "AWS_Lambda_dotnet8", "Unknown"},
		{"Go Runtime", "AWS_Lambda_go1.x", "Unknown"},
		{"Unknown Runtime", "AWS_Lambda_unknown", "Unknown"},
		{"Empty Env", "", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalStatFunc := osStatFunc
			originalReadDirFunc := osReadDirFunc
			defer func() {
				osStatFunc = originalStatFunc
				osReadDirFunc = originalReadDirFunc
			}()

			osStatFunc = mockStatNotFound
			osReadDirFunc = mockReadDirEmpty

			originalExecEnv := os.Getenv("AWS_EXECUTION_ENV")
			originalHandler := os.Getenv("_HANDLER")
			originalRuntimeDir := os.Getenv("LAMBDA_RUNTIME_DIR")
			originalRuntimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API")
			originalTaskRoot := os.Getenv("LAMBDA_TASK_ROOT")
			defer func() {
				restoreEnv("AWS_EXECUTION_ENV", originalExecEnv)
				restoreEnv("_HANDLER", originalHandler)
				restoreEnv("LAMBDA_RUNTIME_DIR", originalRuntimeDir)
				restoreEnv("AWS_LAMBDA_RUNTIME_API", originalRuntimeAPI)
				restoreEnv("LAMBDA_TASK_ROOT", originalTaskRoot)
			}()

			os.Unsetenv("AWS_EXECUTION_ENV")
			os.Unsetenv("_HANDLER")
			os.Unsetenv("LAMBDA_RUNTIME_DIR")
			os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
			os.Unsetenv("LAMBDA_TASK_ROOT")

			if tt.envValue != "" {
				os.Setenv("AWS_EXECUTION_ENV", tt.envValue)
			}

			result := DetectRuntime()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectRuntimeFromHandler(t *testing.T) {
	tests := []struct {
		name     string
		handler  string
		expected string
	}{
		{"Node JS Handler", "index.js", "Node"},
		{"Node MJS Handler", "app.mjs", "Node"},
		{"Node CJS Handler", "handler.cjs", "Node"},
		{"Python Handler", "lambda_function.py", "Unknown"},
		{"Ruby Handler", "lambda_function.rb", "Unknown"},
		{"Java Handler with ::", "com.example.Handler::handleRequest", "Unknown"},
		{"Java JAR Handler", "app.jar", "Unknown"},
		{"Dotnet DLL Handler", "app.dll", "Unknown"},
		{"Unknown Handler", "unknown.txt", "Unknown"},
		{"Empty Handler", "", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalStatFunc := osStatFunc
			originalReadDirFunc := osReadDirFunc
			defer func() {
				osStatFunc = originalStatFunc
				osReadDirFunc = originalReadDirFunc
			}()

			osStatFunc = mockStatNotFound
			osReadDirFunc = mockReadDirEmpty

			originalExecEnv := os.Getenv("AWS_EXECUTION_ENV")
			originalHandler := os.Getenv("_HANDLER")
			originalRuntimeDir := os.Getenv("LAMBDA_RUNTIME_DIR")
			originalRuntimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API")
			originalTaskRoot := os.Getenv("LAMBDA_TASK_ROOT")
			defer func() {
				restoreEnv("AWS_EXECUTION_ENV", originalExecEnv)
				restoreEnv("_HANDLER", originalHandler)
				restoreEnv("LAMBDA_RUNTIME_DIR", originalRuntimeDir)
				restoreEnv("AWS_LAMBDA_RUNTIME_API", originalRuntimeAPI)
				restoreEnv("LAMBDA_TASK_ROOT", originalTaskRoot)
			}()

			os.Unsetenv("AWS_EXECUTION_ENV")
			os.Unsetenv("_HANDLER")
			os.Unsetenv("LAMBDA_RUNTIME_DIR")
			os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
			os.Unsetenv("LAMBDA_TASK_ROOT")

			if tt.handler != "" {
				os.Setenv("_HANDLER", tt.handler)
			}

			result := DetectRuntime()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectRuntimeFromTaskRoot(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "task_root_test")
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		fileName string
		expected string
	}{
		{"Package.json", "package.json", "Node"},
		{"Requirements.txt", "requirements.txt", "Unknown"},
		{"Gemfile", "Gemfile", "Unknown"},
		{"Pom.xml", "pom.xml", "Unknown"},
		{"Build.gradle", "build.gradle", "Unknown"},
		{"Go.mod", "go.mod", "Unknown"},
		{"Project.csproj", "MyProject.csproj", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalStatFunc := osStatFunc
			originalReadDirFunc := osReadDirFunc
			defer func() {
				osStatFunc = originalStatFunc
				osReadDirFunc = originalReadDirFunc
			}()

			filePath := filepath.Join(tempDir, tt.fileName)
			file, err := os.Create(filePath)
			assert.Nil(t, err)
			file.Close()

			osStatFunc = func(path string) (os.FileInfo, error) {
				if path == filePath {
					return os.Stat(path)
				}
				return nil, errors.New("file not found")
			}

			if strings.HasSuffix(tt.fileName, ".csproj") {
				osReadDirFunc = func(path string) ([]os.DirEntry, error) {
					if path == tempDir {
						return os.ReadDir(path)
					}
					return []os.DirEntry{}, nil
				}
			} else {
				osReadDirFunc = mockReadDirEmpty
			}

			originalExecEnv := os.Getenv("AWS_EXECUTION_ENV")
			originalHandler := os.Getenv("_HANDLER")
			originalRuntimeDir := os.Getenv("LAMBDA_RUNTIME_DIR")
			originalRuntimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API")
			originalTaskRoot := os.Getenv("LAMBDA_TASK_ROOT")
			defer func() {
				restoreEnv("AWS_EXECUTION_ENV", originalExecEnv)
				restoreEnv("_HANDLER", originalHandler)
				restoreEnv("LAMBDA_RUNTIME_DIR", originalRuntimeDir)
				restoreEnv("AWS_LAMBDA_RUNTIME_API", originalRuntimeAPI)
				restoreEnv("LAMBDA_TASK_ROOT", originalTaskRoot)
			}()

			os.Unsetenv("AWS_EXECUTION_ENV")
			os.Unsetenv("_HANDLER")
			os.Unsetenv("LAMBDA_RUNTIME_DIR")
			os.Setenv("AWS_LAMBDA_RUNTIME_API", "localhost:9001")
			os.Setenv("LAMBDA_TASK_ROOT", tempDir)

			result := DetectRuntime()
			assert.Equal(t, tt.expected, result)

			os.Remove(filePath)
		})
	}
}

func TestDetectRuntimeAllMethodsFail(t *testing.T) {
	originalStatFunc := osStatFunc
	originalReadDirFunc := osReadDirFunc
	defer func() {
		osStatFunc = originalStatFunc
		osReadDirFunc = originalReadDirFunc
	}()

	osStatFunc = mockStatNotFound
	osReadDirFunc = mockReadDirEmpty

	originalExecEnv := os.Getenv("AWS_EXECUTION_ENV")
	originalHandler := os.Getenv("_HANDLER")
	originalRuntimeDir := os.Getenv("LAMBDA_RUNTIME_DIR")
	originalRuntimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	originalTaskRoot := os.Getenv("LAMBDA_TASK_ROOT")
	defer func() {
		restoreEnv("AWS_EXECUTION_ENV", originalExecEnv)
		restoreEnv("_HANDLER", originalHandler)
		restoreEnv("LAMBDA_RUNTIME_DIR", originalRuntimeDir)
		restoreEnv("AWS_LAMBDA_RUNTIME_API", originalRuntimeAPI)
		restoreEnv("LAMBDA_TASK_ROOT", originalTaskRoot)
	}()

	os.Unsetenv("AWS_EXECUTION_ENV")
	os.Unsetenv("_HANDLER")
	os.Unsetenv("LAMBDA_RUNTIME_DIR")
	os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
	os.Unsetenv("LAMBDA_TASK_ROOT")

	result := DetectRuntime()
	assert.Equal(t, "Unknown", result)
}

func TestDetectRuntimeFromRuntimeDir(t *testing.T) {
	tests := []struct {
		name       string
		runtimeDir string
		expected   string
	}{
		{"Node Runtime Dir", "/var/runtime/nodejs20.x", "Node"},
		{"Java Runtime Dir", "/var/runtime/java21", "Unknown"},
		{"Ruby Runtime Dir", "/var/runtime/ruby3.2", "Unknown"},
		{"Dotnet Runtime Dir", "/var/runtime/dotnet8", "Unknown"},
		{"Go Runtime Dir", "/var/runtime/go1.x", "Unknown"},
		{"Unknown Runtime Dir", "/var/runtime/unknown", "Unknown"},
		{"Empty Runtime Dir", "", "Unknown"},
		{"Case Insensitive Python", "/VAR/RUNTIME/PYTHON3.11", "Unknown"},
		{"Case Insensitive Node", "/VAR/RUNTIME/NODEJS18.X", "Node"},
		{"Case Insensitive Java", "/VAR/RUNTIME/JAVA17", "Unknown"},
		{"Case Insensitive Ruby", "/VAR/RUNTIME/RUBY3.0", "Unknown"},
		{"Case Insensitive Dotnet", "/VAR/RUNTIME/DOTNET6", "Unknown"},
		{"Case Insensitive Go", "/VAR/RUNTIME/GO1.19", "Unknown"},
		{"Partial Match Python", "/some/path/python/lib", "Unknown"},
		{"Partial Match Node", "/some/path/nodejs/bin", "Node"},
		{"Partial Match Java", "/some/path/java/runtime", "Unknown"},
		{"Partial Match Ruby", "/some/path/ruby/gems", "Unknown"},
		{"Partial Match Dotnet", "/some/path/dotnet/shared", "Unknown"},
		{"Partial Match Go", "/some/path/go/bin", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalStatFunc := osStatFunc
			originalReadDirFunc := osReadDirFunc
			defer func() {
				osStatFunc = originalStatFunc
				osReadDirFunc = originalReadDirFunc
			}()

			osStatFunc = mockStatNotFound
			osReadDirFunc = mockReadDirEmpty

			originalExecEnv := os.Getenv("AWS_EXECUTION_ENV")
			originalHandler := os.Getenv("_HANDLER")
			originalRuntimeDir := os.Getenv("LAMBDA_RUNTIME_DIR")
			originalRuntimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API")
			originalTaskRoot := os.Getenv("LAMBDA_TASK_ROOT")
			defer func() {
				restoreEnv("AWS_EXECUTION_ENV", originalExecEnv)
				restoreEnv("_HANDLER", originalHandler)
				restoreEnv("LAMBDA_RUNTIME_DIR", originalRuntimeDir)
				restoreEnv("AWS_LAMBDA_RUNTIME_API", originalRuntimeAPI)
				restoreEnv("LAMBDA_TASK_ROOT", originalTaskRoot)
			}()

			os.Unsetenv("AWS_EXECUTION_ENV")
			os.Unsetenv("_HANDLER")
			os.Unsetenv("LAMBDA_RUNTIME_DIR")
			os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
			os.Unsetenv("LAMBDA_TASK_ROOT")

			if tt.runtimeDir != "" {
				os.Setenv("LAMBDA_RUNTIME_DIR", tt.runtimeDir)
			}

			result := DetectRuntime()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func restoreEnv(key, value string) {
	if value != "" {
		os.Setenv(key, value)
	} else {
		os.Unsetenv(key)
	}
}
