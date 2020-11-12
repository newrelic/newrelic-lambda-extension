package logserver

import (
	"encoding/json"
	"fmt"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	platformLogBufferSize = 100
	defaultHost           = "sandbox"
	fallbackHost          = "169.254.79.130"
)

type LogLine struct {
	RequestID string
	Content   []byte
}

type LogServer struct {
	listenString    string
	server          *http.Server
	platformLogChan chan LogLine
	functionLogChan chan []LogLine
}

func (ls *LogServer) Port() uint16 {
	_, portStr, _ := net.SplitHostPort(ls.listenString)
	port, _ := strconv.ParseUint(portStr, 10, 16)
	return uint16(port)
}

func (ls *LogServer) Close() error {
	// Pause briefly to allow final platform logs to arrive
	time.Sleep(200 * time.Millisecond)

	close(ls.platformLogChan)
	return ls.server.Close()
}

func (ls *LogServer) PollPlatformChannel() []LogLine {
	var ret []LogLine
	for {
		select {
		case report, more := <-ls.platformLogChan:
			if more {
				ret = append(ret, report)
			} else {
				return ret
			}
		default:
			return ret
		}
	}
}

func (ls *LogServer) AwaitFunctionLogs() []LogLine {
	return <- ls.functionLogChan
}

func formatReport(metrics map[string]interface{}) string {
	ret := ""

	if val, ok := metrics["durationMs"]; ok {
		ret += fmt.Sprintf("\tDuration: %.2f ms", val)
	}

	if val, ok := metrics["billedDurationMs"]; ok {
		ret += fmt.Sprintf("\tBilled Duration: %.0f ms", val)
	}

	if val, ok := metrics["memorySizeMB"]; ok {
		ret += fmt.Sprintf("\tMemory Size: %.0f MB", val)
	}

	if val, ok := metrics["maxMemoryUsedMB"]; ok {
		ret += fmt.Sprintf("\tMax Memory Used: %.0f MB", val)
	}

	if val, ok := metrics["initDurationMs"]; ok {
		ret += fmt.Sprintf("\tInit Duration: %.2f ms", val)
	}

	return ret
}

func (ls *LogServer) handler(res http.ResponseWriter, req *http.Request) {
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		util.Logf("Error processing log request: %v", err)
	}

	var logEvents []api.LogEvent
	err = json.Unmarshal(bodyBytes, &logEvents)
	if err != nil {
		util.Logf("Error parsing log payload: %v", err)
	}

	var functionLogs []LogLine
	var lastRequestId string
	for _, event := range logEvents {
		switch event.Type {
		case "platform.start":
			lastRequestId = event.Record.(map[string]string)["requestId"]
		case "platform.report":
			record := event.Record.(map[string]interface{})
			metrics := record["metrics"].(map[string]interface{})
			requestId := record["requestId"].(string)
			reportStr := fmt.Sprintf(
				"REPORT RequestId: %v%s",
				requestId,
				formatReport(metrics),
			)
			reportLine := LogLine{
				RequestID: requestId,
				Content:   []byte(reportStr),
			}
			ls.platformLogChan <- reportLine
		case "platform.logsDropped":
			util.Logf("Platform dropped logs: %v", event.Record)
		//TODO: handle function logs. NB: they should send directly, as they don't need to decorate telemetry
		case "function":
			record := event.Record.(string)
			functionLogs = append(functionLogs, LogLine{
				RequestID: lastRequestId,
				Content:   []byte(record),
			})
		default:
			log.Println("Ignored log event of type ", event.Type, string(bodyBytes))
		}
	}
	if len(functionLogs) > 0 {
		ls.functionLogChan <- functionLogs
	}

	_, _ = res.Write(nil)
}

func Start() (*LogServer, error) {
	return startInternal(defaultHost)
}

func startInternal(host string) (*LogServer, error) {
	// TODO: replace fallbackHost with host and remove the fallback in the error handler once 'sandbox' resolves.
	listener, err := net.Listen("tcp", fallbackHost+":")
	if err != nil {
		listener, err = net.Listen("tcp", host+":")
		if err != nil {
			return nil, err
		}
	}

	server := http.Server{}

	logServer := LogServer{
		listenString:    listener.Addr().String(),
		server:          &server,
		platformLogChan: make(chan LogLine, platformLogBufferSize),
		functionLogChan: make(chan []LogLine),
	}

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		logServer.handler(res, req)
	})

	go func() {
		util.Logln("Starting log server.")
		util.Logf("Log server terminating: %v\n", server.Serve(listener))
	}()

	return &logServer, nil
}
