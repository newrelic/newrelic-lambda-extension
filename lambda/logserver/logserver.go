package logserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

const (
	platformLogBufferSize = 100
)

type LogLine struct {
	Time      time.Time
	RequestID string
	Content   []byte
}

type LogServer struct {
	listenString      string
	server            *http.Server
	platformLogChan   chan LogLine
	functionLogChan   chan []LogLine
	lastRequestId     string
	lastRequestIdLock *sync.Mutex
	isShuttingDown    bool
	shutdownLock      sync.RWMutex

	wg                sync.WaitGroup
}

func (ls *LogServer) Port() uint16 {
	_, portStr, _ := net.SplitHostPort(ls.listenString)
	port, _ := strconv.ParseUint(portStr, 10, 16)
	return uint16(port)
}

func (ls *LogServer) Close() error {

	ls.shutdownLock.Lock()
	ls.isShuttingDown = true
	ls.shutdownLock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ret := ls.server.Shutdown(ctx)
	if ret == context.DeadlineExceeded {
		ret = nil
	}
	ls.wg.Wait()
	close(ls.platformLogChan)
	close(ls.functionLogChan)
	return ret
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

func (ls *LogServer) AwaitFunctionLogs() ([]LogLine, bool) {
	ll, more := <-ls.functionLogChan
	return ll, more
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
	util.Debugf("Formatted Return Report: %s", ret)
	return ret
}

var reportStringRegExp, _ = regexp.Compile("RequestId: ([a-fA-F0-9-]+)(.*)")

func (ls *LogServer) handler(res http.ResponseWriter, req *http.Request) {
	defer util.Close(req.Body)
	ls.wg.Add(1)
	defer ls.wg.Done()

	ls.shutdownLock.RLock()
	logServerShuttingDown := ls.isShuttingDown
	ls.shutdownLock.RUnlock()
	if logServerShuttingDown {
		_, _ = res.Write(nil)
		return
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		util.Logf("Error processing log request: %v", err)
	}

	var logEvents []api.LogEvent
	err = json.Unmarshal(bodyBytes, &logEvents)
	if err != nil {
		util.Logf("Error parsing log payload: %v", err)
	}

	var functionLogs []LogLine

	for _, event := range logEvents {
		switch event.Type {
		case "platform.start":
			ls.lastRequestIdLock.Lock()
			switch event.Record.(type) {
			case map[string]interface{}:
				ls.lastRequestId = event.Record.(map[string]interface{})["requestId"].(string)
			case string:
				recordString := event.Record.(string)
				results := reportStringRegExp.FindStringSubmatch(recordString)
				if len(results) > 1 {
					ls.lastRequestId = results[1]
				}
			}
			ls.lastRequestIdLock.Unlock()
		case "platform.report":
			metricString := ""
			requestId := ""
			switch event.Record.(type) {
			case map[string]interface{}:
				record := event.Record.(map[string]interface{})
				metrics := record["metrics"].(map[string]interface{})
				metricString = formatReport(metrics)
				requestId = record["requestId"].(string)
			case string:
				recordString := event.Record.(string)
				results := reportStringRegExp.FindStringSubmatch(recordString)
				if len(results) > 1 {
					requestId = results[1]
					if len(results) > 2 {
						metricString = results[2]
					}
				} else {
					util.Debugf("Unknown platform log: %s", recordString)
				}
			}

			reportStr := fmt.Sprintf(
				"REPORT RequestId: %v%s",
				requestId,
				metricString,
			)
			reportLine := LogLine{
				Time:      event.Time,
				RequestID: requestId,
				Content:   []byte(reportStr),
			}
			ls.platformLogChan <- reportLine
		case "platform.logsDropped":
			util.Logf("Platform dropped logs: %v", event.Record)
		case "function", "extension", "platform.fault":
			record := event.Record.(string)
			ls.lastRequestIdLock.Lock()
			functionLogs = append(functionLogs, LogLine{
				Time:      event.Time,
				RequestID: ls.lastRequestId,
				Content:   []byte(record),
			})
			ls.lastRequestIdLock.Unlock()
		default:
			//util.Debugln("Ignored log event of type ", event.Type, string(bodyBytes))
		}
	}

	if len(functionLogs) > 0 {
		ls.functionLogChan <- functionLogs
	}

	_, _ = res.Write(nil)
}

func Start(conf *config.Configuration) (*LogServer, error) {
	return startInternal(conf.LogServerHost)
}

func startInternal(host string) (*LogServer, error) {
	listener, err := net.Listen("tcp", host+":")
	if err != nil {
		return nil, err
	}

	server := &http.Server{}

	logServer := &LogServer{
		listenString:      listener.Addr().String(),
		server:            server,
		platformLogChan:   make(chan LogLine, platformLogBufferSize),
		functionLogChan:   make(chan []LogLine),
		lastRequestIdLock: &sync.Mutex{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", logServer.handler)
	server.Handler = mux

	go func() {
		util.Logln("Starting log server.")
		util.Logf("Log server started to terminate: %v\n", server.Serve(listener))
	}()

	return logServer, nil
}
