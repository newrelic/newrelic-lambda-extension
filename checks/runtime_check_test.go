//go:build !race
// +build !race

package checks

import (
	"os"
	"path/filepath"
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

func TestDetectRuntimeFromAWSExecutionEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{"Python Runtime", "AWS_Lambda_python3.13", "Python"},
		{"Node Runtime", "AWS_Lambda_nodejs20.x", "Node"},
		{"Java Runtime", "AWS_Lambda_java21", "Java"},
		{"Ruby Runtime", "AWS_Lambda_ruby3.2", "Ruby"},
		{"Dotnet Runtime", "AWS_Lambda_dotnet8", "Dotnet"},
		{"Go Runtime", "AWS_Lambda_go1.x", "Go"},
		{"Unknown Runtime", "AWS_Lambda_unknown", "Unknown"},
		{"Empty Env", "", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env
			originalEnv := os.Getenv("AWS_EXECUTION_ENV")
			defer os.Setenv("AWS_EXECUTION_ENV", originalEnv)

			// Set test env
			if tt.envValue != "" {
				os.Setenv("AWS_EXECUTION_ENV", tt.envValue)
			} else {
				os.Unsetenv("AWS_EXECUTION_ENV")
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
		{"Python Handler", "lambda_function.py", "Python"},
		{"Ruby Handler", "lambda_function.rb", "Ruby"},
		{"Java Handler with ::", "com.example.Handler::handleRequest", "Java"},
		{"Java JAR Handler", "app.jar", "Java"},
		{"Dotnet DLL Handler", "app.dll", "Dotnet"},
		{"Unknown Handler", "unknown.txt", "Unknown"},
		{"Empty Handler", "", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear AWS_EXECUTION_ENV to test handler detection
			originalExecEnv := os.Getenv("AWS_EXECUTION_ENV")
			originalHandler := os.Getenv("_HANDLER")
			defer func() {
				os.Setenv("AWS_EXECUTION_ENV", originalExecEnv)
				os.Setenv("_HANDLER", originalHandler)
			}()

			os.Unsetenv("AWS_EXECUTION_ENV")
			if tt.handler != "" {
				os.Setenv("_HANDLER", tt.handler)
			} else {
				os.Unsetenv("_HANDLER")
			}

			result := DetectRuntime()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectRuntimeFromRuntimeDir(t *testing.T) {
	tests := []struct {
		name       string
		runtimeDir string
		expected   string
	}{
		{"Python Runtime Dir", "/var/runtime/python3.13", "Python"},
		{"Node Runtime Dir", "/var/runtime/nodejs20.x", "Node"},
		{"Java Runtime Dir", "/var/runtime/java21", "Java"},
		{"Ruby Runtime Dir", "/var/runtime/ruby3.2", "Ruby"},
		{"Dotnet Runtime Dir", "/var/runtime/dotnet8", "Dotnet"},
		{"Go Runtime Dir", "/var/runtime/go1.x", "Go"},
		{"Unknown Runtime Dir", "/var/runtime/unknown", "Unknown"},
		{"Empty Runtime Dir", "", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalExecEnv := os.Getenv("AWS_EXECUTION_ENV")
			originalHandler := os.Getenv("_HANDLER")
			originalRuntimeDir := os.Getenv("LAMBDA_RUNTIME_DIR")
			defer func() {
				os.Setenv("AWS_EXECUTION_ENV", originalExecEnv)
				os.Setenv("_HANDLER", originalHandler)
				os.Setenv("LAMBDA_RUNTIME_DIR", originalRuntimeDir)
			}()

			os.Unsetenv("AWS_EXECUTION_ENV")
			os.Unsetenv("_HANDLER")
			if tt.runtimeDir != "" {
				os.Setenv("LAMBDA_RUNTIME_DIR", tt.runtimeDir)
			} else {
				os.Unsetenv("LAMBDA_RUNTIME_DIR")
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
		{"Requirements.txt", "requirements.txt", "Python"},
		{"Gemfile", "Gemfile", "Ruby"},
		{"Pom.xml", "pom.xml", "Java"},
		{"Build.gradle", "build.gradle", "Java"},
		{"Go.mod", "go.mod", "Go"},
		{"Project.csproj", "MyProject.csproj", "Dotnet"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalExecEnv := os.Getenv("AWS_EXECUTION_ENV")
			originalHandler := os.Getenv("_HANDLER")
			originalRuntimeDir := os.Getenv("LAMBDA_RUNTIME_DIR")
			originalRuntimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API")
			originalTaskRoot := os.Getenv("LAMBDA_TASK_ROOT")
			defer func() {
				os.Setenv("AWS_EXECUTION_ENV", originalExecEnv)
				os.Setenv("_HANDLER", originalHandler)
				os.Setenv("LAMBDA_RUNTIME_DIR", originalRuntimeDir)
				os.Setenv("AWS_LAMBDA_RUNTIME_API", originalRuntimeAPI)
				os.Setenv("LAMBDA_TASK_ROOT", originalTaskRoot)
			}()

			os.Unsetenv("AWS_EXECUTION_ENV")
			os.Unsetenv("_HANDLER")
			os.Unsetenv("LAMBDA_RUNTIME_DIR")
			os.Setenv("AWS_LAMBDA_RUNTIME_API", "localhost:9001")
			os.Setenv("LAMBDA_TASK_ROOT", tempDir)

			filePath := filepath.Join(tempDir, tt.fileName)
			file, err := os.Create(filePath)
			assert.Nil(t, err)
			file.Close()

			result := DetectRuntime()
			assert.Equal(t, tt.expected, result)

			os.Remove(filePath)
		})
	}
}

func TestDetectRuntimePriority(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "priority_test")
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	originalExecEnv := os.Getenv("AWS_EXECUTION_ENV")
	originalHandler := os.Getenv("_HANDLER")
	originalTaskRoot := os.Getenv("LAMBDA_TASK_ROOT")
	originalRuntimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	defer func() {
		os.Setenv("AWS_EXECUTION_ENV", originalExecEnv)
		os.Setenv("_HANDLER", originalHandler)
		os.Setenv("LAMBDA_TASK_ROOT", originalTaskRoot)
		os.Setenv("AWS_LAMBDA_RUNTIME_API", originalRuntimeAPI)
	}()

	os.Setenv("AWS_EXECUTION_ENV", "AWS_Lambda_python3.13")
	os.Setenv("_HANDLER", "index.js")
	os.Setenv("LAMBDA_TASK_ROOT", tempDir)
	os.Setenv("AWS_LAMBDA_RUNTIME_API", "localhost:9001")

	packageJsonPath := filepath.Join(tempDir, "package.json")
	file, err := os.Create(packageJsonPath)
	assert.Nil(t, err)
	file.Close()

	result := DetectRuntime()
	assert.Equal(t, "Python", result, "AWS_EXECUTION_ENV should take priority")
}

func TestDetectRuntimeAllMethodsFail(t *testing.T) {
	originalExecEnv := os.Getenv("AWS_EXECUTION_ENV")
	originalHandler := os.Getenv("_HANDLER")
	originalRuntimeDir := os.Getenv("LAMBDA_RUNTIME_DIR")
	originalRuntimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	originalTaskRoot := os.Getenv("LAMBDA_TASK_ROOT")
	defer func() {
		os.Setenv("AWS_EXECUTION_ENV", originalExecEnv)
		os.Setenv("_HANDLER", originalHandler)
		os.Setenv("LAMBDA_RUNTIME_DIR", originalRuntimeDir)
		os.Setenv("AWS_LAMBDA_RUNTIME_API", originalRuntimeAPI)
		os.Setenv("LAMBDA_TASK_ROOT", originalTaskRoot)
	}()

	os.Unsetenv("AWS_EXECUTION_ENV")
	os.Unsetenv("_HANDLER")
	os.Unsetenv("LAMBDA_RUNTIME_DIR")
	os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
	os.Unsetenv("LAMBDA_TASK_ROOT")

	result := DetectRuntime()
	assert.Equal(t, "Unknown", result)
}
