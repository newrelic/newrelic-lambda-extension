package apm

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type harvest struct {
	data [][]byte
}

type apmConfig struct {
	Configuration *config.Configuration
	metadata map[string]string
	hostname string
	LambdaFunctionName string
	LambdaFunctionVersion string
	LambdaAccountId string
}

type harvestable interface {
	MergeIntoHarvest(h *harvest)
}

type InternalAPMApp struct {
	rpmControls    rpmControls
	apmConfig      apmConfig
	apmHarvest     *harvest

	initiateShutdown chan time.Duration
	shutdownStarted  chan struct{}
	shutdownComplete chan struct{}

	DataChan           chan []byte
	ErrorEventChan	   chan interface{}
	collectorErrorChan chan rpmResponse
	connectChan        chan *appRun
	LambdaLogChan      chan string

	// This mutex protects both `run` and `err`, both of which should only
	// be accessed using getState and setState.
	sync.RWMutex
	// run is non-nil when the app is successfully connected.  It is
	// immutable.
	Run *appRun
	// err is non-nil if the application will never be connected again
	// (disconnect, license exception, shutdown).
	err error
}

func NewApp(ctx context.Context ,c *config.Configuration, LambdaFunctionName string, LambdaAccountId string, LambdaFunctionVersion string) *InternalAPMApp {
	app := &InternalAPMApp{
		apmConfig:         apmConfig{
			Configuration: c,
			LambdaFunctionName: LambdaFunctionName,
			LambdaFunctionVersion: LambdaFunctionVersion,
			LambdaAccountId: LambdaAccountId,},

		initiateShutdown: make(chan time.Duration, 1),

		shutdownStarted:    make(chan struct{}),
		shutdownComplete:   make(chan struct{}),
		apmHarvest:         &harvest{data: [][]byte{}},	 
		connectChan:        make(chan *appRun, 1),
		collectorErrorChan: make(chan rpmResponse, 1),
		DataChan:           make(chan []byte, 5),
		LambdaLogChan:      make(chan string, 1),
		rpmControls: rpmControls{
			License: c.LicenseKey,
			Client: &http.Client{
				Timeout: 20 * time.Second,
			},
			GzipWriterPool: &sync.Pool{
				New: func() interface{} {
					return gzip.NewWriter(io.Discard)
				},
			},
		},
	}
	util.Debugf("New Internl APM app created with serverless config")
	go app.process(ctx)
	go app.connectRoutine()

	return app
}

func (app *InternalAPMApp) connectAttempt() (*ConnectReply, *rpmResponse) {
	preconnectCollectorHost := preconnectHost(app.apmConfig.Configuration)

	cmd := RpmCmd{
		Name: cmdPreconnect,
		Collector: preconnectCollectorHost,
		metaData: map[string]interface{}{
			"AWSFunctionName": app.apmConfig.LambdaFunctionName,
			"AWSAccountId": app.apmConfig.LambdaAccountId,
			"AWSFunctionVersion": app.apmConfig.LambdaFunctionVersion,
		},
	}
	redirectHost, PreConnectErr := PreConnect(cmd, &app.rpmControls)
	if PreConnectErr != nil {
		return nil, &rpmResponse{err: PreConnectErr}
	}
	app.apmConfig.hostname = redirectHost
	cmd.Collector = redirectHost
	cmd.Name = cmdConnect
	runId, entityGuid, ConnectErr := Connect(cmd, &app.rpmControls)
	if ConnectErr != nil {
		return nil, &rpmResponse{err: ConnectErr}
	}

	return &ConnectReply{
		RunID:      runId,
		EntityGUID: entityGuid,
	}, nil
}

func (app *InternalAPMApp) connectRoutine() {
	attempts := 0
	maxAttempts := 3

	for attempts < maxAttempts {
		reply, resp := app.connectAttempt()
		if reply != nil {
			util.Debugf("Connect successful in attempt %d", attempts+1)

			select {
			case app.connectChan <- newAppRun(app.apmConfig, reply):
			case <-app.shutdownStarted:
			}
			return
		}

		if resp.IsDisconnect() {
			select {
			case app.collectorErrorChan <- *resp:
			case <-app.shutdownStarted:
			}
			return
		}

		if nil != resp.GetError() {
			util.Debugf("Error connecting to collector: %v", resp.GetError())
		}

		backoff := getConnectBackoffTime(attempts)
		time.Sleep(time.Duration(backoff) * time.Second)
		attempts++
	}

	util.Debugf("Exceeded maximum connection attempts.")
	util.Fatal(fmt.Errorf("failed to connect to collector after %d attempts", maxAttempts))
}

func getConnectBackoffTime(attempt int) int {
	connectBackoffTimes := [...]int{15, 15, 30}
	l := len(connectBackoffTimes)
	if (attempt < 0) || (attempt >= l) {
		return connectBackoffTimes[l-1]
	}
	return connectBackoffTimes[attempt]
}

func (app *InternalAPMApp) setState(run *appRun, err error) {
	app.Lock()
	defer app.Unlock()

	app.Run = run
	app.err = err
}


func (app *InternalAPMApp) doHarvest(ctx context.Context, payload []byte, run *appRun) {
	collectorHost := app.apmConfig.hostname 
	util.Debugf("Harvest collector host: %s", collectorHost)
	cmd := RpmCmd{
		Name: cmdPreconnect,
		Collector: collectorHost,
		metaData: map[string]interface{}{
			"AWSFunctionName": app.apmConfig.LambdaFunctionName,
			"AWSAccountId": app.apmConfig.LambdaAccountId,
			"AWSFunctionVersion": app.apmConfig.LambdaFunctionVersion,
		},
	}

	runId := run.Reply.RunID

	resp, _ := SendAPMTelemetry(ctx, payload, app.apmConfig.Configuration, cmd, &app.rpmControls, runId)
	if resp != nil {
		util.Debugf("Error sending telemetry data: %v", resp)
		nrResponse := newRPMResponse(resp)
		app.collectorErrorChan <- *nrResponse
		util.Debugf("Error sent to collector error channel")
	}
	util.Debugf("Harvest sent to collector")
}

func (app *InternalAPMApp) process(ctx context.Context) {
	var run *appRun
	util.Debugf("Starting APM process loop....")

	for {
		select {
		case data := <-app.DataChan:
			if nil != run && run.Reply.RunID != "" {
				util.Debugf("Received data in DataChan with length: %d", len(data))
				go app.doHarvest(ctx, data, run)
				if app.apmHarvest != nil {
					util.Debugf("Harvesting data from DataChan")
					for _, harvestableData := range app.apmHarvest.data {
						util.Debugf("Harvesting data: %s", string(harvestableData))
						go app.doHarvest(ctx, harvestableData, run)
					}
				}
			} else {
				util.Debugf("Received data in DataChan but runId not available, saving data for later")
				app.apmHarvest.data = append(app.apmHarvest.data, data)
			}
		case resp := <-app.collectorErrorChan:
			util.Debugf("Received error in CollectorErrorChan: %v", resp)
			app.setState(nil, nil)
			if resp.IsDisconnect() {
				util.Fatal(fmt.Errorf("collector disconnected: %v", resp.GetError()))
			} else if resp.IsRestartException() {
				util.Debugf("Received restart exception, resetting app state")
				go app.connectRoutine()
			}
		case <-app.shutdownStarted:
			util.Debugf("Shutdown started")
			return
		case run = <-app.connectChan:
			util.Debugf("Received run in ConnectChan and setting app state")
			app.setState(run, nil)
			app.LambdaLogChan <- run.Reply.EntityGUID
		case <-app.ErrorEventChan:
			util.Debugf("Received error event in ErrorEventChan")
		}
	}
}
