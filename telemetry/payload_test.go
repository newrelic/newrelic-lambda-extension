package telemetry

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePayload(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", \"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\"]")

	data, err := parsePayload(payload)

	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestParsePayloadBlank(t *testing.T) {
	payload := []byte("")

	data, err := parsePayload(payload)

	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestParsePayloadInvalidCompressedData(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", \"foobar\"]")

	data, err := parsePayload(payload)

	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestParsePayloadInvalidData(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", \"H4sIAK6pdWIC/0vLz09KLAIAlR/2ngYAAAA=\"]")

	data, err := parsePayload(payload)

	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestExtractTraceIDAnalyticEvent(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", \"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\"]")
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)

	assert.Nil(t, err)
	assert.Equal(t, "24d071916e1f00ee", traceId)
}

func TestExtractTraceIDSpanEvent(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", \"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\"]")
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)

	assert.Nil(t, err)
	assert.Equal(t, "446cf2064d931f4e", traceId)
}

func TestExtractTraceIDInvalid(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", \"H4sIAK6pdWIC/0vLz09KLAIAlR/2ngYAAAA=\"]")
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)

	assert.Error(t, err)
	assert.Empty(t, traceId)

	payload2 := []byte("[foobar]")
	payload2 = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err = ExtractTraceID(payload2)

	assert.Nil(t, err)
	assert.Empty(t, traceId)

	payload3 := []byte("[foobar]")

	traceId, err = ExtractTraceID(payload3)

	assert.Error(t, err)
	assert.Empty(t, traceId)
}

func TestExtractTraceIDPayloadVersion2AnalyticEvent(t *testing.T) {
	payload := []byte(`[2,"NR_LAMBDA_MONITORING",{"metadata_version":2,"arn":"arn:aws:lambda:us-east-1:466768951184:function:go-ext-test-HelloWorldFunction-OzWnlEPQ09ra","protocol_version":17,"agent_version":"3.36.0","agent_language":"go"},"H4sIAAAAAAAA/+xWS3PbNhD+L3sGSfBN4tRXmvTQSTpxmoNH41kRKwoTEGAB0K7s0X/vUKITO7UTu1WnPfSg0WoB7bevjx9vAA3qXVDdBV2SCRcSA4I4B2A34MiTu7TKXXh1TSBSzjlncLjoLzyRAZHu2fn5DYTdSCDgzKHx2AVlDTAwOMzO12FL7s5J8tImvY3o9xAF8iF6RVrb99Zp+eNkDjei19fvjX7x5hfeOgQGQQ3kAw4jiLTO26apyqZp65wBOWcdiA1qTwzk5PCALXjMy7Zp2qptGAQbUJ+pOZl7/n5SEgQgpXlJVVHxkktqD4AOO/rpocOad3xTIpcoi5byNTAYnbJOhR2INM7qrM1zBh6HUZMEEdxEe3YD3eSDHb4Nwan1FOa2fOb5FfVEMF/FKx9rHNYS485q+TagC8dADDorKdbKkLEgsiJbPJulcSBgNlNY/PME/IjdjNersJ3WcWeHBK/8/ImOKFFvkwXP0UZTF16hkZrcbZSN0jRi2IKA5J0n5xPES2XQb5PeJuOHPhmsTL4U/5vLNC7SmC84yfYIEPcW2KFeR79N5MOx5VWTFZksoyLLqqhosIrapkwjWRZYFGsqs3qz/G1JG91cOToj8MqLo1NMPiL0IUpFUVV11bRlmjaFuG2VeM4K7lerFVsm9gSi5F8lys+7l/bFfPzIdrc8z+dduFYjCLjGcd6MYxYDBae6O/ifKPHJbA9gJyLgnp2n7C5z+P1f922el01V12WbVWWeZjyvVyv2hWRQ638W4ewj/f8M8zT7rwL+/UafJL8flsfid7vvUWtyyTvzwdgr8+j3wyP513I59PekCc088iOaJ3A5+xqV345onisZi/JsOlnLfE3rqmvzNu2OYW7X6MFg8Jm2PCw/jwvmIxJ5MqnuMFBv3W4WHDLk1FyUcTGZ4HZvrDIfpexOpfHJHlTPEtr/xW0Z2n/njWOWuP0fAAAA//8BAAD//8swp8OVCgAA"]`)
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)

	assert.Nil(t, err)
	assert.Equal(t, "ae135e646050de9a70c0f5a0dad49e3b", traceId)
}

func TestExtractTraceIDPayloadVersion2SpanEvent(t *testing.T) {
	payload := []byte("[2,\"NR_LAMBDA_MONITORING\",{\"metadata_version\":2,\"arn\":\"arn:aws:lambda:us-east-1:466768951184:function:go-ext-test-HelloWorldFunction-OzWnlEPQ09ra\",\"protocol_version\":17,\"agent_version\":\"3.36.0\",\"agent_language\":\"go\"},\"H4sIAG2gtWcC/+1X21LbMBB95ysYP8f3q/LUG4U+dKADlAeGyaytjePBllJZDk2Y/HtlJ6SkxCYtGWg7ZCaJIp3dlc5Zbda3e/vq1XxoSVVKXgxwgkwOKEjQ+vuXzVKzrPVW49vVqFkRWKKY8EwMymyGysq11Ku3DmrcloMSkSmEvVqc/8Rdrlms/3oYduVZTsd1UO3z9JAf1FG0XgswK7CUUIzrDYQuiaLAj4jlug/g8962wWdZ7U2bwVjbyst8bepqb310tbDQCpQiS7pFWJ0g2jBHnswqg6Jh9ViOUJwJYCUkMuPMPORmynX8LnWp2NSPMM/5BRc5/VixBqEfzy5YfnDyxSICtmLlcuMW7M0yWoblq0OSgEQtgD+12wJguX4UhKFPnMB3bcdywwfIq3WFezumH/L8ldPdcnrGJeRnqji8BLHPAvh7KP4Hysf/qciHSkBN77vpe1VCUJjn7JrxG9b6/UKF5pX9O/abO/QqwWO9UjkGtoum1XmJnvVUbb61XRWQ4CdawwBt18fACyzfokggtBJr6INFgXoE3bjNRVpljf0woSF1Y4yDhLjETjpC3lXszYHbDEvVVedYm0hRYQtoLDIuMjmtuTSc0CGu+zudekTCNjxdXi8FfzT3dtbZtrhPQGLKRX1KLUWGqpFvgzJhqAQT0xOeqaeWTu7uKWM8Z2vekr+LZ8W3Uj2mxJVsdvPL1FfIK3xSCE7RGGY5jkGO6gDmubqxpQkwyRiUI3VUc3ydmgWnZprJURUbCS9MuCnrt55DEVPQU/5mYhuebVjmYsYcAaOqAhspb5NFmRsCv1WKw+UtCCLHc6ive44T6F4EgU4i39ap74Hnxeg74bDL2SKyAYI13gTrq9n+YrZflTqCUsvue0EQBhHxbTvy+sOlZP0dpOS9PSQ8p6cSxCPp1pCfZwwZr0uj53Th7rbaVBo1trUudJ29qmYnTdJ0CbcUTEkxzDGRRwvdNqRU99/E3vwHxv8etOUQAAA=\"]")
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)

	assert.Nil(t, err)
	assert.Equal(t, "ae135e646050de9a70c0f5a0dad49e3b", traceId)
}

func TestParsePayloadVersion2InvalidData(t *testing.T) {
	payload := []byte("[2, \"NR_LAMBDA_MONITORING\", \"H4sIAK6pdWIC/0vLz09KLAIAlR/2ngYAAAA=\"]")

	data, err := parsePayload(payload)

	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestExtractTraceIDPayloadVersion2Invalid(t *testing.T) {
	payload := []byte("[2, \"NR_LAMBDA_MONITORING\",{\"metadata_version\":2,\"arn\":\"arn:aws:lambda:us-east-1:466768951184:function:go-ext-test-HelloWorldFunction-OzWnlEPQ09ra\",\"protocol_version\":17,\"agent_version\":\"3.36.0\",\"agent_language\":\"go\"}, \"H4sIAK6pdWIC/0vLz09KLAIAlR/2ngYAAAA=\"]")
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)

	assert.Error(t, err)
	assert.Empty(t, traceId)

	payload2 := []byte("[foobar]")
	payload2 = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err = ExtractTraceID(payload2)

	assert.Nil(t, err)
	assert.Empty(t, traceId)

	payload3 := []byte("[foobar]")

	traceId, err = ExtractTraceID(payload3)

	assert.Error(t, err)
	assert.Empty(t, traceId)
}

func TestParsePayloadVersion2InvalidVersion(t *testing.T) {
	payload := []byte("[\"invalid\", \"NR_LAMBDA_MONITORING\", \"foo\", \"H4sIAK6pdWIC/0vLz09KLAIAlR/2ngYAAAA=\"]")

	data, err := parsePayload(payload)

	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestParsePayloadVersion2InvalidJSON(t *testing.T) {
	payload := []byte("[\"2\", \"NR_LAMBDA_MONITORING\", \"foo\", \"H4sIAK6pdWIC/0vLz09KLAIAlR/2ngYAAAA=")
	data, err := parsePayload(payload)
	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestParsePayloadVersion2InvalidCompressedData(t *testing.T) {
	payload := []byte("[\"2\", \"NR_LAMBDA_MONITORING\", \"foo\", \"invalid_base64\"]")

	data, err := parsePayload(payload)
	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestParsePayloadVersion2EmptyCompressedData(t *testing.T) {
	payload := []byte("[\"2\", \"NR_LAMBDA_MONITORING\", \"foo\", \"\"]")

	data, err := parsePayload(payload)
	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestParsePayloadVersion2MalformedUncompressedJSON(t *testing.T) {
	payload := []byte("[\"2\", \"NR_LAMBDA_MONITORING\", \"foo\", \"H4sIAK6pdWIC/6tmqGZkYKhmBGIglgsZGRgYaxlqGVgYaxhqGBsaGpkam1qYGBibmhkZmlgYGhobG5iYmhgbW5oZGFsYmhgYGxuamRgbmVoZGpkZmxqYGBsZW5gaGRhaGpgYmpgaGBoaGhgaGpkaGxkbGRsaGhsYGllYGBpaWBoZGhuYGlpamBgYmBpamBsbGZkamFgYGFiamxoZGVkaGJmaWJgZGBoYGFha/K9lqGWoZQQA5H6Q4DUAAAA=\"]")

	data, err := parsePayload(payload)
	assert.Nil(t, data)
	assert.Error(t, err)
}
func TestExtractTraceIDVersion1AnalyticEventEmpty(t *testing.T) {
	payload := []byte(`[1, "NR_LAMBDA_MONITORING", "H4sIAAAAAAAA/6pWykxRslLKKlYqLs3NTSyqVLIqKSpN1VEqLU4t8kxRsjI0MTG3NDI3MjAwMDSt5aoFAAAA//8BAAD//3cqHKg0AAAA"]`)
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)
	assert.Error(t, err)
	assert.Empty(t, traceId)
}

func TestExtractTraceIDVersion1SpanEventEmpty(t *testing.T) {
	payload := []byte(`[1, "NR_LAMBDA_MONITORING", "H4sIAAAAAAAA/6pWKi7NzU0sqlSyKikqTdVRKi1OLfJMUbIyNDExtzQyNzIwMDA0rQUEAAD//wEAAP//V7DEFEQAAAA="]`)
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)
	assert.Error(t, err)
	assert.Empty(t, traceId)
}

func TestExtractTraceIDVersion2AnalyticEventEmpty(t *testing.T) {
	payload := []byte(`[2, "NR_LAMBDA_MONITORING", {"metadata_version":2}, "H4sIAAAAAAAA/6pWKi7NzU0sqlSyKikqTdVRKi1OLfJMUbIyNDExtzQyNzIwMDA0rQUEAAD//wEAAP//V7DEFEQAAAA="]`)
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)
	assert.Error(t, err)
	assert.Empty(t, traceId)
}

func TestExtractTraceIDVersion2SpanEventEmpty(t *testing.T) {
	payload := []byte(`[2, "NR_LAMBDA_MONITORING", {"metadata_version":2}, "H4sIAAAAAAAA/6pWKi7NzU0sqlSyKikqTdVRKi1OLfJMUbIyNDExtzQyNzIwMDA0rQUEAAD//wEAAP//V7DEFEQAAAA="]`)
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)
	assert.Error(t, err)
	assert.Empty(t, traceId)
}

func TestExtractTraceIDVersion1MalformedAnalyticEvent(t *testing.T) {
	payload := []byte(`[1, "NR_LAMBDA_MONITORING", "H4sIAAAAAAAA/6pWKi7NzU0sqlSyKikqTdVRKi1OLfJMUbIyNDExtzQyNzIwMDA0rQUA{invalid}"]`)
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)
	assert.Error(t, err)
	assert.Empty(t, traceId)
}

func TestExtractTraceIDVersion2MalformedSpanEvent(t *testing.T) {
	payload := []byte(`[2, "NR_LAMBDA_MONITORING", {"metadata_version":2}, "H4sIAAAAAAAA/6pWKi7NzU0sqlSyKikqTdVRKi1OLfJMUbIyNDExtzQyNzIwMDA0rQUA{invalid}"]`)
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)
	assert.Error(t, err)
	assert.Empty(t, traceId)
}

func TestExtractTraceIDVersion1NonLambdaMonitoring(t *testing.T) {
	payload := []byte(`[1, "SOME_OTHER_EVENT", "H4sIAAAAAAAA/6pWKi7NzU0sqlSyKikqTdVRKi1OLfJMUbIyNDExtzQyNzIwMDA0rQUEAAD//wEAAP//V7DEFEQAAAA="]`)
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)
	assert.Nil(t, err)
	assert.Empty(t, traceId)
}

func TestExtractTraceIDVersion2NonLambdaMonitoring(t *testing.T) {
	payload := []byte(`[2, "SOME_OTHER_EVENT", {"metadata_version":2}, "H4sIAAAAAAAA/6pWKi7NzU0sqlSyKikqTdVRKi1OLfJMUbIyNDExtzQyNzIwMDA0rQUEAAD//wEAAP//V7DEFEQAAAA="]`)
	payload = []byte(base64.StdEncoding.EncodeToString(payload))

	traceId, err := ExtractTraceID(payload)
	assert.Nil(t, err)
	assert.Empty(t, traceId)
}
