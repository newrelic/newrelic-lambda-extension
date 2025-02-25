package checks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
)

var testHandler = "path/to/app.handler"

func TestRuntimeMethods(t *testing.T) {
	conf := config.Configuration{TestingOverride: true}
	r := runtimeConfigs[Node]
	h := handlerConfigs{
		handlerName: r.wrapperName,
		conf:        &conf,
	}
	conf.NRHandler = testHandler

	t1 := r.getTrueHandler(h)
	t2 := removePathMethodName(t1)
	t3 := pathFormatter(t2, r.fileType)

	e1 := testHandler
	e2 := "path/to/app"
	e3 := "/var/task/path/to/app.js"

	assert.Equal(t, e1, t1)
	assert.Equal(t, e2, t2)
	assert.Equal(t, e3, t3)

	r = runtimeConfigs[Python]

	h = handlerConfigs{
		handlerName: r.wrapperName,
		conf:        &conf,
	}

	t1 = r.getTrueHandler(h)
	t2 = removePathMethodName(t1)
	t3 = pathFormatter(t2, r.fileType)

	e1 = testHandler
	e2 = "path/to/app"
	e3 = "/var/task/path/to/app.py"

	assert.Equal(t, e1, t1)
	assert.Equal(t, e2, t2)
	assert.Equal(t, e3, t3)
}

func TestHandlerCheckJS(t *testing.T) {
	conf := config.Configuration{TestingOverride: true}
	reg := api.RegistrationResponse{}
	r := runtimeConfigs[Node]
	ctx := context.Background()

	// No Runtime
	err := handlerCheck(ctx, &conf, &reg, runtimeConfig{})
	assert.Nil(t, err)

	// Error
	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.EqualError(t, err, "missing handler file path/to/app.handler (NEW_RELIC_LAMBDA_HANDLER=Undefined)")

	// Success
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	handlerPath = filepath.Join(dirname, "var", "task")
	os.MkdirAll(filepath.Join(handlerPath, "path", "to"), os.ModePerm)
	os.Create(filepath.Join(handlerPath, "path", "to", "app.js"))

	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.Nil(t, err)
}

func TestHandlerCheckMJS(t *testing.T) {
	conf := config.Configuration{TestingOverride: true}
	reg := api.RegistrationResponse{}
	r := runtimeConfigs[Node]
	ctx := context.Background()

	// No Runtime
	err := handlerCheck(ctx, &conf, &reg, runtimeConfig{})
	assert.Nil(t, err)

	// Error
	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.EqualError(t, err, "missing handler file path/to/app.handler (NEW_RELIC_LAMBDA_HANDLER=Undefined)")

	// Success
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	handlerPath = filepath.Join(dirname, "var", "task")
	os.MkdirAll(filepath.Join(handlerPath, "path", "to"), os.ModePerm)
	os.Create(filepath.Join(handlerPath, "path", "to", "app.mjs"))

	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.Nil(t, err)
}

func TestHandlerCheckCJS(t *testing.T) {
	conf := config.Configuration{TestingOverride: true}
	reg := api.RegistrationResponse{}
	r := runtimeConfigs[Node]
	ctx := context.Background()

	// No Runtime
	err := handlerCheck(ctx, &conf, &reg, runtimeConfig{})
	assert.Nil(t, err)

	// Error
	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.EqualError(t, err, "missing handler file path/to/app.handler (NEW_RELIC_LAMBDA_HANDLER=Undefined)")

	// Success
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	handlerPath = filepath.Join(dirname, "var", "task")
	os.MkdirAll(filepath.Join(handlerPath, "path", "to"), os.ModePerm)
	os.Create(filepath.Join(handlerPath, "path", "to", "app.cjs"))

	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.Nil(t, err)
}

func TestHandlerCheckPython(t *testing.T) {
	conf := config.Configuration{TestingOverride: true}
	reg := api.RegistrationResponse{}
	r := runtimeConfigs[Python]
	ctx := context.Background()

	// No Runtime
	err := handlerCheck(ctx, &conf, &reg, runtimeConfig{})
	assert.Nil(t, err)

	// Error
	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.EqualError(t, err, "missing handler file path/to/app.handler (NEW_RELIC_LAMBDA_HANDLER=Undefined)")

	// Success
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	handlerPath = filepath.Join(dirname, "var", "task")
	os.MkdirAll(filepath.Join(handlerPath, "path", "to"), os.ModePerm)
	os.Create(filepath.Join(handlerPath, "path", "to", "app.py"))

	reg.Handler = testHandler
	conf.NRHandler = config.EmptyNRWrapper
	err = handlerCheck(ctx, &conf, &reg, r)
	assert.Nil(t, err)
}

func TestRemovePathMethodName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path with method",
			input:    "index.handler",
			expected: "index",
		},
		{
			name:     "nested path with method",
			input:    "src/handlers/index.handler",
			expected: "src/handlers/index",
		},
		{
			name:     "multiple dots in path",
			input:    "src.test.index.handler",
			expected: "src/test/index",
		},
		{
			name:     "no method name",
			input:    "index",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removePathMethodName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemovePathMethodNameNode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple node path",
			input:    "index.handler",
			expected: "index",
		},
		{
			name:     "nested node path",
			input:    "src/handlers/index.handler",
			expected: "src/handlers/index",
		},
		{
			name:     "path with multiple dots",
			input:    "src/index.test.handler",
			expected: "src/index",
		},
		{
			name:     "path with special characters",
			input:    "src/my-handler.test.handler",
			expected: "src/my-handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removePathMethodNameNode(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandlerCheck(t *testing.T) {
	tests := []struct {
		name            string
		envVars         map[string]string
		testingOverride bool
		language        string
		handler         string
		nrHandler       string
		wrapperName     string
		expectError     bool
	}{
		{
			name: "ESM enabled",
			envVars: map[string]string{
				"NEW_RELIC_USE_ESM": "true",
			},
			testingOverride: false,
			language:        "nodejs",
			handler:         "index.handler",
			expectError:     false,
		},
		{
			name: "Docker environment",
			envVars: map[string]string{
				"AWS_EXECUTION_ENV": "local",
			},
			testingOverride: false,
			language:        "nodejs",
			handler:         "index.handler",
			expectError:     false,
		},
		{
			name:            "Node handler with JS file",
			testingOverride: true,
			language:        "nodejs",
			handler:         "index.handler",
			nrHandler:       "src/index.handler",
			wrapperName:     "newrelic-lambda-wrapper",
			expectError:     true,
		},
		{
			name:            "Non-node handler",
			testingOverride: true,
			language:        "python",
			handler:         "handler.handle",
			nrHandler:       "src/handler.handle",
			wrapperName:     "newrelic-lambda-wrapper",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			conf := &config.Configuration{
				TestingOverride: tt.testingOverride,
				NRHandler:       tt.nrHandler,
			}

			reg := &api.RegistrationResponse{
				Handler: tt.handler,
			}

			r := runtimeConfig{
				language:    "python",
				wrapperName: tt.wrapperName,
				fileType:    "py",
			}

			err := handlerCheck(context.Background(), conf, reg, r)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPathFormatter(t *testing.T) {
	tests := []struct {
		name            string
		functionHandler string
		fileType        string
		expected        string
	}{
		{
			name:            "simple handler",
			functionHandler: "index",
			fileType:        "js",
			expected:        "/var/task/index.js",
		},
		{
			name:            "nested handler",
			functionHandler: "src/handlers/index",
			fileType:        "py",
			expected:        "/var/task/src/handlers/index.py",
		},
		{
			name:            "handler with dashes",
			functionHandler: "my-handler",
			fileType:        "mjs",
			expected:        "/var/task/my-handler.mjs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pathFormatter(tt.functionHandler, tt.fileType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDocker(t *testing.T) {
	originalEnv := os.Getenv("AWS_EXECUTION_ENV")
	defer os.Setenv("AWS_EXECUTION_ENV", originalEnv)

	testCases := []struct {
		envValue string
		expected bool
	}{
		{
			envValue: "AWS_Lambda_nodejs14.x",
			expected: false,
		},
		{
			envValue: "Docker",
			expected: true,
		},
		{
			envValue: "",
			expected: true,
		},
	}

	for _, tc := range testCases {
		os.Setenv("AWS_EXECUTION_ENV", tc.envValue)
		result := isDocker()
		assert.Equal(t, tc.expected, result)
	}
}

func TestCheckWithTestingOverride(t *testing.T) {
	originalESM := os.Getenv("NEW_RELIC_USE_ESM")
	defer os.Setenv("NEW_RELIC_USE_ESM", originalESM)
	originalAWS := os.Getenv("AWS_EXECUTION_ENV")
	defer os.Setenv("AWS_EXECUTION_ENV", originalAWS)

	conf := config.Configuration{TestingOverride: false}
	h := handlerConfigs{
		handlerName: "test.handler",
		conf:        &conf,
	}
	r := runtimeConfig{
		language:    Node,
		fileType:    "js",
		wrapperName: "test.handler",
	}

	os.Setenv("NEW_RELIC_USE_ESM", "true")
	result := r.check(h)
	assert.True(t, result)

	os.Setenv("NEW_RELIC_USE_ESM", "false")
	os.Setenv("AWS_EXECUTION_ENV", "Docker")
	result = r.check(h)
	assert.True(t, result)

	conf.TestingOverride = true
	os.Setenv("NEW_RELIC_USE_ESM", "false")
	os.Setenv("AWS_EXECUTION_ENV", "AWS_Lambda_nodejs14.x")

	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	handlerPath = filepath.Join(dirname, "var", "task")
	os.MkdirAll(filepath.Join(handlerPath, "test"), os.ModePerm)
	os.Create(filepath.Join(handlerPath, "test", "handler.js"))

	conf.NRHandler = "test/handler.method"
	result = r.check(h)
	assert.True(t, result)
}
