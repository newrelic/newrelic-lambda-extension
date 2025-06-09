package apm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeUncompress_Success(t *testing.T) {
	original := []byte("hello world")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(original)
	assert.NoError(t, err)
	assert.NoError(t, gz.Close())

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	result, err := decodeUncompress(encoded)
	assert.NoError(t, err)
	assert.Equal(t, original, result)
}

func TestDecodeUncompress_InvalidBase64(t *testing.T) {
	invalidBase64 := "!!!notbase64!!!"
	result, err := decodeUncompress(invalidBase64)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDecodeUncompress_InvalidGzip(t *testing.T) {
	// Valid base64, but not gzipped data
	notGzipped := base64.StdEncoding.EncodeToString([]byte("not gzipped"))
	result, err := decodeUncompress(notGzipped)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDecodeUncompress_EmptyInput(t *testing.T) {
	result, err := decodeUncompress("")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetServerlessData_ProtocolVersion2_Success(t *testing.T) {
	payload := LambdaData{
		MetricData:            []interface{}{"metric"},
		CustomEventData:       []interface{}{"custom"},
		LogEventData:          []interface{}{"log"},
		AnalyticEventData:     []interface{}{"analytic"},
		ErrorEventData:        []interface{}{"error_event"},
		ErrorData:             []interface{}{"error"},
		SpanEventData:         []interface{}{"span"},
		UpdateLoadedModules:   []interface{}{"module"},
		TransactionSampleData: []interface{}{"transaction"},
	}
	jsonBytes, err := json.Marshal(payload)
	assert.NoError(t, err)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(jsonBytes)
	assert.NoError(t, err)
	assert.NoError(t, gz.Close())

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	input := []byte(`[2,"` + encoded + `"]`)

	raw, data, version, err := GetServerlessData(input)
	assert.NoError(t, err)
	assert.Equal(t, 2, version)
	assert.Equal(t, LambdaRawData{}, raw)
	assert.Equal(t, payload, data)
}

func TestGetServerlessData_ProtocolVersion1_Success(t *testing.T) {
	payload := LambdaRawData{
		LambdaMetadata: LambdaMetadata{
			MetadataVersion:      1,
			ARN:                  "arn:aws:lambda:region:account-id:function:function-name",
			ProtocolVersion:      1,
			ExecutionEnvironment: "AWS_Lambda_nodejs12.x",
			AgentVersion:         "1.0.0",
			AgentLanguage:        "go",
		},
		LambdaData: LambdaData{
			MetricData: []interface{}{"metric"},
		},
	}
	jsonBytes, err := json.Marshal(payload)
	assert.NoError(t, err)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(jsonBytes)
	assert.NoError(t, err)
	assert.NoError(t, gz.Close())

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	input := []byte(`[1,"` + encoded + `"]`)

	raw, data, version, err := GetServerlessData(input)
	assert.NoError(t, err)
	assert.Equal(t, 1, version)
	assert.Equal(t, payload, raw)
	assert.Equal(t, LambdaData{}, data)
}


func TestGetServerlessData_EmptyInput(t *testing.T) {
	raw, data, version, err := GetServerlessData([]byte{})
	assert.NoError(t, err)
	assert.Equal(t, 0, version)
	assert.Equal(t, LambdaRawData{}, raw)
	assert.Equal(t, LambdaData{}, data)
}

func TestGetServerlessData_InvalidFormat(t *testing.T) {
	raw, data, version, err := GetServerlessData([]byte(`{"foo":"bar"}`))
	assert.NoError(t, err)
	assert.Equal(t, 0, version)
	assert.Equal(t, LambdaRawData{}, raw)
	assert.Equal(t, LambdaData{}, data)
}

func TestGetServerlessData_InsufficientComponents(t *testing.T) {
	input := []byte(`[2]`)
	_, _, _, err := GetServerlessData(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient data components")
}

func TestGetServerlessData_InvalidBase64OrGzip(t *testing.T) {
	input := []byte(`[2,"!!!notbase64!!!"]`)
	_, _, _, err := GetServerlessData(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode and decompress data")
}

func TestGetServerlessData_InvalidJSONUnmarshal(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte("not json"))
	assert.NoError(t, err)
	assert.NoError(t, gz.Close())
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	input := []byte(`[2,"` + encoded + `"]`)
	_, _, _, err = GetServerlessData(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to unmarshal JSON data into LambdaData")
}
