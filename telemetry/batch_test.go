package telemetry

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/stretchr/testify/assert"
)

const (
	testTelemetry       = "test_telemetry"
	moreTestTelemetry   = "more_test_telemetry"
	testRequestId       = "test_a"
	testRequestId2      = "test_b"
	testRequestId3      = "test_c"
	testNoSuchRequestId = "test_z"
	ripe                = 1000
	rot                 = 10000
	isAPMTelemetry      = true
)

var (
	requestStart = time.Unix(1603821157, 0)
)

func generateNLengthTelemetryString(length int) string {
	outStr := ""
	for i := 0; i < length; i++ {
		outStr += "a"
	}

	return outStr
}

func TestMissingInvocation(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	invocation := batch.AddTelemetry(testNoSuchRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.Nil(t, invocation)
}

func TestEmptyHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot, false)
	res := batch.Harvest(requestStart)

	assert.Nil(t, res)
}

func TestEmptyRotHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.AddInvocation("test", requestStart)

	res := batch.Harvest(requestStart)

	assert.Empty(t, res)
}

func TestEmptyRipeHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.lastHarvest = requestStart.Add(-ripe)
	batch.AddInvocation("test", requestStart)

	res := batch.Harvest(requestStart)

	assert.Empty(t, res)
}

func TestWithInvocationRipeHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.lastHarvest = requestStart

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, []byte(testTelemetry), isAPMTelemetry)
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)

	harvested := batch.Harvest(requestStart.Add(ripe*time.Millisecond + time.Millisecond))
	assert.Equal(t, 1, len(harvested))
	assert.Equal(t, testRequestId, harvested[0].RequestId)
	assert.Equal(t, 2, len(harvested[0].Telemetry))
}

func TestWithInvocationAggressiveHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)

	harvested := batch.Harvest(requestStart.Add(ripe*time.Millisecond + time.Millisecond))
	assert.Equal(t, 2, len(harvested))
}

func TestBatch_Close(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)

	harvested := batch.Close()
	assert.Equal(t, 2, len(harvested))
}

func TestBatchAsync(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.lastHarvest = requestStart

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		batch.AddInvocation(testRequestId, requestStart)
		wg.Done()
	}()
	go func() {
		batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
		wg.Done()
	}()
	go func() {
		batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))
		wg.Done()
	}()

	// Doing this to try to trigger a panic
	go batch.RetrieveTraceID(testRequestId)

	wg.Wait()

	var invocation, invocation2 *Invocation
	wg.Add(2)

	go func() {
		invocation = batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
		wg.Done()
	}()
	go func() {
		invocation2 = batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes(), isAPMTelemetry)
		wg.Done()
	}()

	// Doing this to try to trigger a panic
	go batch.RetrieveTraceID(testRequestId)

	wg.Wait()
	assert.NotNil(t, invocation)
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)

	harvested := batch.Harvest(requestStart.Add(ripe*time.Millisecond + time.Millisecond))
	go assert.Equal(t, 1, len(harvested))
	go assert.Equal(t, testRequestId, harvested[0].RequestId)
	go assert.Equal(t, 2, len(harvested[0].Telemetry))
}

func TestBatchSetTraceIDValue(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	requestId := "testRequestId"
	expectedTraceID := "testTraceID"
	batch.SetTraceIDValue(requestId, expectedTraceID)

	assert.Equal(t, batch.storeTraceID[requestId], expectedTraceID)
}
func TestBatchRetrieveTraceID(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	requestId := "testRequestId"
	expectedTraceID := "testTraceID"
	batch.SetTraceIDValue(requestId, expectedTraceID)

	traceID := batch.RetrieveTraceID(requestId)
	assert.Equal(t, expectedTraceID, traceID)
	batch.SetTraceIDValue(requestId, "")
	traceID = batch.RetrieveTraceID(requestId)
	assert.Empty(t, traceID)

	nonExistentRequestId := "nonExistentRequestId"
	traceID = batch.RetrieveTraceID(nonExistentRequestId)
	assert.Equal(t, "", traceID)
}
func TestAddTelemetry(t *testing.T) {
	batch := NewBatch(ripe, rot, true)

	batch.AddInvocation(testRequestId, requestStart)
	inv := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.NotNil(t, inv)
	assert.Equal(t, 1, len(inv.Telemetry))
	assert.Equal(t, testTelemetry, string(inv.Telemetry[0]))

	inv2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes(), isAPMTelemetry)
	assert.NotNil(t, inv2)
	assert.Equal(t, 2, len(inv2.Telemetry))
	assert.Equal(t, moreTestTelemetry, string(inv2.Telemetry[1]))

	assert.Equal(t, requestStart, batch.eldest)

	traceId := "testTraceId"
	encodedTelemetry := base64.StdEncoding.EncodeToString([]byte(traceId))

	inv3 := batch.AddTelemetry(testRequestId, []byte(encodedTelemetry), isAPMTelemetry)
	assert.NotNil(t, inv3)
	assert.Equal(t, "", inv3.TraceId)
	assert.Equal(t, "", batch.RetrieveTraceID(testRequestId))

	inv4 := batch.AddTelemetry(testNoSuchRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.Nil(t, inv4)

	inv5 := batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes(), !isAPMTelemetry)
	assert.Nil(t, inv5)
}
func TestAddTelemetryWithTraceIDExtraction(t *testing.T) {
	telemetryRequestId := "a89efeea-261f-47c1-8d7d-250e40ad9670"
	isAPMTelemetry := true
	expectedTraceID := "b3c694dab92e4f9ecbd26656926214c6"
	data := []interface{}{1, "NR_LAMBDA_MONITORING", "H4sIAEUlq2cC/+VYa2/bNhT9K4Wwj7ZEUhIl+luatmiHFAvqZB0QBAItMY5WWVRJKqkb5L/vUrJjJ5Zdx3a7YIMhW+bj8PDecy8fd85EGJ5xw53BnVMpaWQqi+RGKJ3L0hngqOeIbyKtDfxNRHmTK1lORGmcgXP0eZic8Mko40k1Ndey9F1MnJ7Dx1C/gHAwcqmLHioKXo5reIWatputUbYhfA/4rR4UDeig1n3BtenjQUBpRGMWYhwHg6u6TC2bQdu7bxRPRZ71fUz6RkD796Io5GepiuzdrGmfi9uT0ft3H7/+dR3CcHOIJZK/nRydvR2eOfc9Z26NuoI3kRSSZyJLJjKrC6GdwYXzO1fa6V1cOEC25aryFGCJS+w87+4vexeOzidVIf7WzfzAMsz151WqLk0+EUla5NaSTzqW4laJogGcW+5xhXst5RftTgqgJIpEfymEtV/PefW0ZdKygyrkYjdeA5RKJcG/pdAJ19MyzWUX1qxxIcdjoRL7k5fjDQ3FNyNUyYvk2piqyEcPTS8vwd9QMTVAUNxYSbQWvyjroujdOUpooW5krhKdfweZYIIQqNC21IkWwsryHsx/54zrPAPfjfyUsiDjI0ZEcMUEDKW5tT5UGlWLHgg7lyo3U+jpEkbjgPacRjcfuvqno4xQGlJGKMFBSgHP+ksbALUh4UMFCwI/CGNQS624aTSEXESQlWgUMIp8EuEQOk4rq/QzxUvNG9EBWskntvAPcy3UUo0316u3u7SNNLw4yy1+B597cABYGHTrKvG1BsjGADxm4koI3icUX/WDKMX9OIuyPgmRCBDPGI2a+IVuraLcXxywSyPP0f5cCd1HzSCLZUPDlWkl0MpOV7zcSnKguDWSMwt3fVijvWcJaxuhthIaAvmFdg4glVn0+FEorhiiLESMjYK91J5CwhxLBfQdyPWiTYtLJnN/ovJL5YK71PRU5uXc6/9vucPSDh54JPWFO92AEAROXRSFLqEoApqN0GeeGtZVJZXho7wAWXqnzZy8owpSetoowfskxrk2rSy8N3N9gLJT2egWltQL3LNy8X2EKcIB6AwTvHcZQmEAMRJQEkNUBTiGOe9M/MgYMamMXiH+8NmEftIuh947qW65yuzrbEBR8pGN771gT2TKizcCFmogu8DOcn0A8I+NSvRhCfORKPSOPH9GclgV49PctWcZQgEKotCHkIoI8zEK6OZJ8aJ4ibzO5vuH/6TVH2b3EsjNc+Xr6TGIQSjvvPxSytty7e8LkcwuvBtHvATy3cvCJ5tbs+XwtFPTnmrKu3gjRGBNYjFBMcUxpgGihykOXAabsShkFPuxnYroo21Wtvdc3UBUzohfFeJbDjl3hbrvshiHDJEIVlKKWADw4QFKsRvGYHWKYSqIoSiIoJg94n18eu6dwz771boAfPp09vbODcz7+9pdxg9BhlMNS/3uJNr+e9NoUtHuLJrue5H4KCawU/dOr6cadkSrmcUPXUbCKMYk3OodE4ZcSnEUxrC1jENMNw33A+o4jkBCIWaUwbkkZPGeZcgPScCwj3Ho+5j6bCM39i8ao2PwX22ODyUc/spUQFq0iaa953lCaf7ZIjMdS1glUiOV13mj5v1Rm6o23usp7CpWDU/IowcjH07Fzxp1A77fw5SS9suiR74dATEcka1WjeZMOWyO8MqbHTF3PEJ0Y7bXA/tBLizRce222foUxY8enzI4aT1rzCd3LpvHs/ZffkI/jMnG8eytyFuL3mbE5lUP7Y3Nc/PhdsDNfe1ewC2St7TZeKsUGKqDNDoc9grvXbGPa23kRByU7xLm/jw7zuKHoNoJuwPbDpw3SlbVyvF4DdTl5f39P0toy3q2GQAA"}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Error marshalling data: %v", err)
	}

	byteArray := []byte(jsonData)
	conf := config.ConfigurationFromEnvironment()
	conf.CollectTraceID = true
	batch := NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)

	batch.AddInvocation("a89efeea-261f-47c1-8d7d-250e40ad9670", time.Now())

	inv := batch.invocations[telemetryRequestId]
	if conf.CollectTraceID && isAPMTelemetry {
		telemetryBytesEncoded := []byte(base64.StdEncoding.EncodeToString(byteArray))
		traceId, err := ExtractTraceID(telemetryBytesEncoded)
		if err != nil {
			assert.NotNil(t, err)
		}
		if traceId != "" {
			inv.TraceId = traceId
			assert.Equal(t, inv.TraceId, expectedTraceID)
			batch.SetTraceIDValue(telemetryRequestId, traceId)
			assert.Equal(t, batch.RetrieveTraceID(telemetryRequestId), expectedTraceID)
		}
	}

}

func TestAddTelemetry2(t *testing.T) {
	conf := config.ConfigurationFromEnvironment()
	conf.CollectTraceID = true
	batch := NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)
	batch.extractTraceID = true
	expectedTraceID := "b3c694dab92e4f9ecbd26656926214c6"
	data := []interface{}{1, "NR_LAMBDA_MONITORING", "H4sIAEUlq2cC/+VYa2/bNhT9K4Wwj7ZEUhIl+luatmiHFAvqZB0QBAItMY5WWVRJKqkb5L/vUrJjJ5Zdx3a7YIMhW+bj8PDecy8fd85EGJ5xw53BnVMpaWQqi+RGKJ3L0hngqOeIbyKtDfxNRHmTK1lORGmcgXP0eZic8Mko40k1Ndey9F1MnJ7Dx1C/gHAwcqmLHioKXo5reIWatputUbYhfA/4rR4UDeig1n3BtenjQUBpRGMWYhwHg6u6TC2bQdu7bxRPRZ71fUz6RkD796Io5GepiuzdrGmfi9uT0ft3H7/+dR3CcHOIJZK/nRydvR2eOfc9Z26NuoI3kRSSZyJLJjKrC6GdwYXzO1fa6V1cOEC25aryFGCJS+w87+4vexeOzidVIf7WzfzAMsz151WqLk0+EUla5NaSTzqW4laJogGcW+5xhXst5RftTgqgJIpEfymEtV/PefW0ZdKygyrkYjdeA5RKJcG/pdAJ19MyzWUX1qxxIcdjoRL7k5fjDQ3FNyNUyYvk2piqyEcPTS8vwd9QMTVAUNxYSbQWvyjroujdOUpooW5krhKdfweZYIIQqNC21IkWwsryHsx/54zrPAPfjfyUsiDjI0ZEcMUEDKW5tT5UGlWLHgg7lyo3U+jpEkbjgPacRjcfuvqno4xQGlJGKMFBSgHP+ksbALUh4UMFCwI/CGNQS624aTSEXESQlWgUMIp8EuEQOk4rq/QzxUvNG9EBWskntvAPcy3UUo0316u3u7SNNLw4yy1+B597cABYGHTrKvG1BsjGADxm4koI3icUX/WDKMX9OIuyPgmRCBDPGI2a+IVuraLcXxywSyPP0f5cCd1HzSCLZUPDlWkl0MpOV7zcSnKguDWSMwt3fVijvWcJaxuhthIaAvmFdg4glVn0+FEorhiiLESMjYK91J5CwhxLBfQdyPWiTYtLJnN/ovJL5YK71PRU5uXc6/9vucPSDh54JPWFO92AEAROXRSFLqEoApqN0GeeGtZVJZXho7wAWXqnzZy8owpSetoowfskxrk2rSy8N3N9gLJT2egWltQL3LNy8X2EKcIB6AwTvHcZQmEAMRJQEkNUBTiGOe9M/MgYMamMXiH+8NmEftIuh947qW65yuzrbEBR8pGN771gT2TKizcCFmogu8DOcn0A8I+NSvRhCfORKPSOPH9GclgV49PctWcZQgEKotCHkIoI8zEK6OZJ8aJ4ibzO5vuH/6TVH2b3EsjNc+Xr6TGIQSjvvPxSytty7e8LkcwuvBtHvATy3cvCJ5tbs+XwtFPTnmrKu3gjRGBNYjFBMcUxpgGihykOXAabsShkFPuxnYroo21Wtvdc3UBUzohfFeJbDjl3hbrvshiHDJEIVlKKWADw4QFKsRvGYHWKYSqIoSiIoJg94n18eu6dwz771boAfPp09vbODcz7+9pdxg9BhlMNS/3uJNr+e9NoUtHuLJrue5H4KCawU/dOr6cadkSrmcUPXUbCKMYk3OodE4ZcSnEUxrC1jENMNw33A+o4jkBCIWaUwbkkZPGeZcgPScCwj3Ho+5j6bCM39i8ao2PwX22ODyUc/spUQFq0iaa953lCaf7ZIjMdS1glUiOV13mj5v1Rm6o23usp7CpWDU/IowcjH07Fzxp1A77fw5SS9suiR74dATEcka1WjeZMOWyO8MqbHTF3PEJ0Y7bXA/tBLizRce222foUxY8enzI4aT1rzCd3LpvHs/ZffkI/jMnG8eytyFuL3mbE5lUP7Y3Nc/PhdsDNfe1ewC2St7TZeKsUGKqDNDoc9grvXbGPa23kRByU7xLm/jw7zuKHoNoJuwPbDpw3SlbVyvF4DdTl5f39P0toy3q2GQAA"}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Error marshalling data: %v", err)
	}
	telemetryData := []byte(jsonData)
	requestId := "a89efeea-261f-47c1-8d7d-250e40ad9670"
	isAPMTelemetry := true

	requestStart := time.Now()
	batch.invocations[requestId] = &Invocation{
		Start:     requestStart,
		RequestId: requestId,
		Telemetry: [][]byte{},
	}

	inv := batch.AddTelemetry(requestId, telemetryData, isAPMTelemetry)

	assert.NotNil(t, inv)

	assert.Equal(t, 1, len(inv.Telemetry))
	assert.Equal(t, telemetryData, inv.Telemetry[0])

	assert.Equal(t, requestStart, batch.eldest)

	assert.Equal(t, expectedTraceID, inv.TraceId)

	storedTraceId, exists := batch.storeTraceID[requestId]
	assert.True(t, exists)
	assert.Equal(t, expectedTraceID, storedTraceId)

}
