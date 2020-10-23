package logserver

import (
	"encoding/json"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
)

type LogServer struct {
	ListenString string
	Server *http.Server
}

func (ls *LogServer) Port () uint16 {
	_, portStr, _ := net.SplitHostPort(ls.ListenString)
	port, _ := strconv.ParseUint(portStr, 10, 16)
	return uint16(port)
}

func (ls *LogServer) Close() error {
	return ls.Server.Close()
}

func (ls *LogServer) handler(res http.ResponseWriter, req *http.Request) {
	log.Println("Log Handler Invoked")
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("Error processing log request: %v", err)
	}

	var logEvents []api.LogEvent
	err = json.Unmarshal(bodyBytes, &logEvents)
	if err != nil {
		log.Printf("Error parsing log payload: %v", err)
	}

	for _, event := range logEvents {
		switch event.Type {
		case "platform.extension":
			log.Printf("%v", event.Record)
		case "platform.report":
			//TODO: REPORT line
			log.Printf("ReportViaLogsApi: %v", event.Record)
		case "platform.logsDropped":
			log.Printf("Platform dropped logs: %v", event.Record)
		default:
		}
	}

	_, _ = res.Write(nil)
}

func Start() (*LogServer, error) {
	listener, err := net.Listen("tcp", ":")
	if err != nil {
		return nil, err
	}

	server := http.Server{}

	logServer := LogServer{
		ListenString: listener.Addr().String(),
		Server:       &server,
	}

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		log.Println("Invoking handler for request ", req.RequestURI)
		logServer.handler(res, req)
	})

	go func() {
		log.Println("Starting log server.")
		log.Printf("Log server terminating: %v\n", server.Serve(listener))
	}()

	return &logServer, nil
}
