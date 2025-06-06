package apm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/checks"
	"github.com/stretchr/testify/assert"
)

func TestGetAgentVersion_Node_Success(t *testing.T) {
	tmpDir := t.TempDir()
	packageJSONPath := filepath.Join(tmpDir, "package.json")
	expectedVersion := "1.2.3"
	packageJSON := checks.LayerAgentVersion{Version: expectedVersion}
	b, err := json.Marshal(packageJSON)
	assert.NoError(t, err)
	err = os.WriteFile(packageJSONPath, b, 0644)
	assert.NoError(t, err)

	origPaths := checks.LayerAgentPathNode
	checks.LayerAgentPathNode = []string{tmpDir}
	defer func() { checks.LayerAgentPathNode = origPaths }()

	lang, version, err := getAgentVersion("node")
	assert.NoError(t, err)
	assert.Equal(t, "nodejs", lang)
	assert.Equal(t, expectedVersion, version)
}

func TestGetAgentVersion_Python_Success(t *testing.T) {
	tmpDir := t.TempDir()
	versionTxtPath := filepath.Join(tmpDir, "version.txt")
	expectedVersion := "2.3.4"
	err := os.WriteFile(versionTxtPath, []byte(expectedVersion), 0644)
	assert.NoError(t, err)

	origPaths := checks.LayerAgentPathsPython
	checks.LayerAgentPathsPython = []string{tmpDir}
	defer func() { checks.LayerAgentPathsPython = origPaths }()

	lang, version, err := getAgentVersion("python")
	assert.NoError(t, err)
	assert.Equal(t, "python", lang)
	assert.Equal(t, expectedVersion, version)
}

func TestGetAgentVersion_Node_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	origPaths := checks.LayerAgentPathNode
	checks.LayerAgentPathNode = []string{tmpDir}
	defer func() { checks.LayerAgentPathNode = origPaths }()

	lang, version, err := getAgentVersion("node")
	assert.Error(t, err)
	assert.Equal(t, "", lang)
	assert.Equal(t, "", version)
}

func TestGetAgentVersion_Python_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	origPaths := checks.LayerAgentPathsPython
	checks.LayerAgentPathsPython = []string{tmpDir}
	defer func() { checks.LayerAgentPathsPython = origPaths }()

	lang, version, err := getAgentVersion("python")
	assert.Error(t, err)
	assert.Equal(t, "", lang)
	assert.Equal(t, "", version)
}
func TestGetUtilizationData_ReturnsExpectedData(t *testing.T) {
	os.Setenv("AWS_REGION", "us-west-2")
	defer os.Unsetenv("AWS_REGION")
	os.Unsetenv("AWS_DEFAULT_REGION")

	cmd := RpmCmd{
		metaData: map[string]interface{}{
			"AWSFunctionName": "my-func",
			"AWSAccountId":    "123456789012",
		},
	}

	data := getUtilizationData(cmd)
	vendors, ok := data["vendors"].(map[string]interface{})
	assert.True(t, ok)
	awslambda, ok := vendors["awslambda"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "us-west-2", awslambda["aws.region"])
	assert.Equal(t, "123456789012", awslambda["aws.accountId"])
	assert.Equal(t, "my-func", awslambda["aws.functionName"])
	assert.Contains(t, awslambda["aws.arn"], "arn:aws:lambda:us-west-2:123456789012:function:my-func")
}

func TestGetLambdaARN_ReturnsExpectedARN(t *testing.T) {
	origRegion := os.Getenv("AWS_REGION")
	origDefaultRegion := os.Getenv("AWS_DEFAULT_REGION")
	defer func() {
		os.Setenv("AWS_REGION", origRegion)
		os.Setenv("AWS_DEFAULT_REGION", origDefaultRegion)
	}()

	cmd := RpmCmd{
		metaData: map[string]interface{}{
			"AWSFunctionName": "test-func",
			"AWSAccountId":    "987654321000",
		},
	}

	os.Setenv("AWS_REGION", "eu-central-1")
	os.Unsetenv("AWS_DEFAULT_REGION")
	arn := getLambdaARN(cmd)
	expected := "arn:aws:lambda:eu-central-1:987654321000:function:test-func"
	assert.Equal(t, expected, arn)
}

func TestGetLambdaARN_UsesDefaultRegionIfRegionUnset(t *testing.T) {
	origRegion := os.Getenv("AWS_REGION")
	origDefaultRegion := os.Getenv("AWS_DEFAULT_REGION")
	defer func() {
		os.Setenv("AWS_REGION", origRegion)
		os.Setenv("AWS_DEFAULT_REGION", origDefaultRegion)
	}()

	cmd := RpmCmd{
		metaData: map[string]interface{}{
			"AWSFunctionName": "default-func",
			"AWSAccountId":    "111222333444",
		},
	}

	os.Unsetenv("AWS_REGION")
	os.Setenv("AWS_DEFAULT_REGION", "ap-southeast-2")
	arn := getLambdaARN(cmd)
	expected := "arn:aws:lambda:ap-southeast-2:111222333444:function:default-func"
	assert.Equal(t, expected, arn)
}

func TestCheckRuntime_NodeExists(t *testing.T) {
	origRuntimeLookupPath := runtimeLookupPath
	defer func() { runtimeLookupPath = origRuntimeLookupPath }()

	tmpDir := t.TempDir()
	runtimeLookupPath = tmpDir

	nodePath := filepath.Join(tmpDir, "node")
	err := os.WriteFile(nodePath, []byte{}, 0755)
	assert.NoError(t, err)

	got := checkRuntime()
	assert.Equal(t, NodeLambda, got)
}

func TestCheckRuntime_PythonExists(t *testing.T) {
	origRuntimeLookupPath := runtimeLookupPath
	defer func() { runtimeLookupPath = origRuntimeLookupPath }()

	tmpDir := t.TempDir()
	runtimeLookupPath = tmpDir

	pythonPath := filepath.Join(tmpDir, "python")
	err := os.WriteFile(pythonPath, []byte{}, 0755)
	assert.NoError(t, err)

	got := checkRuntime()
	assert.Equal(t, PythonLambda, got)
}

func TestCheckRuntime_NoneExists_ReturnsDefault(t *testing.T) {
	origRuntimeLookupPath := runtimeLookupPath
	defer func() { runtimeLookupPath = origRuntimeLookupPath }()

	tmpDir := t.TempDir()
	runtimeLookupPath = tmpDir

	got := checkRuntime()
	assert.Equal(t, DefaultLambda, got)
}
