package apm

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
)

var (
	Once sync.Once
	ConnectDone = make(chan struct{})
)

const (
	MaxPayloadSizeInBytes = 1000 * 1000
	procotolVersion = 17
	Version = "3.35.1"

	userAgentPrefix = "NewRelic-Go-Agent/"

	// Methods used in collector communication.
	cmdPreconnect   = "preconnect"
	cmdConnect      = "connect"
	CmdMetrics      = "metric_data"
	CmdCustomEvents = "custom_event_data"
	cmdLogEvents    = "log_event_data"
	cmdTxnEvents    = "analytic_event_data"
	CmdErrorEvents  = "error_event_data"
	CmdErrorData    = "error_data"
	cmdTxnTraces    = "transaction_sample_data"
	cmdSlowSQLs     = "sql_trace_data"
	CmdSpanEvents   = "span_event_data"
)

type RpmCmd struct {
	Name              string
	Collector         string
	RunID             string
	Data              []byte
	RequestHeadersMap map[string]string
	MaxPayloadSize    int
}

type RpmControls struct {
	License        string
	Client         *http.Client
	GzipWriterPool *sync.Pool
	FunctionName   	string
	RunID		  	string
	
}

var (
	mutex    		sync.Mutex
	EntityGuid 		string
)
type rpmResponse struct {
	statusCode  int
	body        []byte
	err         error
}

func SetEntityGuid(entityGuid string) {
	mutex.Lock() 
	defer mutex.Unlock()
	EntityGuid = entityGuid
}

func GetEntityGuid() string {
	mutex.Lock() 
	defer mutex.Unlock()
	return EntityGuid
}


func (cs *RpmControls) SetRunId(runId string) {
	mutex.Lock() 
	defer mutex.Unlock()
	cs.RunID = runId
}

func (cs *RpmControls) GetRunId() string {
	mutex.Lock() 
	defer mutex.Unlock()
	return cs.RunID
}

func RpmURL(cmd RpmCmd, cs *RpmControls) string {
	var u url.URL

	u.Host = cmd.Collector
	u.Path = "agent_listener/invoke_raw_method"
	u.Scheme = "https"

	query := url.Values{}
	query.Set("marshal_format", "json")
	query.Set("protocol_version", strconv.Itoa(procotolVersion))
	query.Set("method", cmd.Name)
	query.Set("license_key", cs.License)

	if len(cmd.RunID) > 0 {
		query.Set("run_id", cmd.RunID)
	}

	u.RawQuery = query.Encode()
	return u.String()
}

// please create all rpmResponses this way
func newRPMResponse(err error) *rpmResponse {
	if err == nil {
		return &rpmResponse{}
	}

	// remove url from errors to avoid sensitive data leaks
	var ue *url.Error
	if errors.As(err, &ue) {
		ue.URL = "**REDACTED-URL**"
	}

	return &rpmResponse{
		err: err,
	}
}

func (resp *rpmResponse) AddStatusCode(statusCode int) *rpmResponse {
	resp.statusCode = statusCode
	if statusCode != 200 && statusCode != 202 {
		resp.err = fmt.Errorf("response code: %d", statusCode)
	}

	return resp
}

func (res *rpmResponse) GetStatusCode() int {
	return res.statusCode
}

// SetError overwrites the existing response error
func (resp *rpmResponse) SetError(err error) *rpmResponse {
	resp.err = err
	return resp
}

// AddBody adds a byte slice containing an http response body
func (resp *rpmResponse) AddBody(body []byte) *rpmResponse {
	resp.body = body
	return resp
}

func (resp *rpmResponse) GetBody() io.Reader {
	return bytes.NewReader(resp.body)
}

func (resp rpmResponse) GetError() error {
	return resp.err
}


// Define structures that match your JSON data
type Metadata struct {
	ProtocolVersion      int    `json:"protocol_version"`
	ExecutionEnvironment string `json:"execution_environment"`
	AgentVersion         string `json:"agent_version"`
	AgentLanguage        string `json:"agent_language"`
	ARN                  string `json:"arn"`
	FunctionVersion      string `json:"function_version"`
}

type Data struct {
	Error_data          []interface{}   `json:"error_data"`
	AnalyticEventData   []interface{}   `json:"analytic_event_data"`
	SpanEventData       []interface{}   `json:"span_event_data"`
	MetricData          []interface{}   `json:"metric_data"`
}

type Input struct {
	Metadata Metadata `json:"metadata"`
	Data     Data     `json:"data"`
}


func compress(b []byte, gzipWriterPool *sync.Pool) (*bytes.Buffer, error) {
	w := gzipWriterPool.Get().(*gzip.Writer)
	defer gzipWriterPool.Put(w)

	var buf bytes.Buffer
	w.Reset(&buf)
	_, err := w.Write(b)
	w.Close()

	if nil != err {
		return nil, err
	}

	return &buf, nil
}

// collectorRequest makes a request to New Relic.
func CollectorRequest(cmd RpmCmd, cs *RpmControls) *rpmResponse {
	url := RpmURL(cmd, cs)
	return collectorRequestInternal(url, cmd, cs)
}

func collectorRequestInternal(url string, cmd RpmCmd, cs *RpmControls) *rpmResponse {
	compressed, err := compress(cmd.Data, cs.GzipWriterPool)
	if err != nil {
		fmt.Printf("Error compressing data: %v", err)
		return newRPMResponse(err)
	}

	req, err := http.NewRequest("POST", url, compressed)
	if err != nil {
		fmt.Printf("Error creating request: %v", err)
		return newRPMResponse(err)
	}

	req.Header.Add("NR-Session", cmd.RunID)
	req.Header.Add("Accept-Encoding", "identity, deflate")
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("Content-Length", strconv.Itoa(len(cmd.Data)))
	req.Header.Add("User-Agent", userAgentPrefix+Version)
	req.Header.Add("Content-Encoding", "gzip")
	for k, v := range cmd.RequestHeadersMap {
		req.Header.Add(k, v)
	}

	resp, err := cs.Client.Do(req)
	if err != nil {
		fmt.Println("Error connecting:", err)
		return newRPMResponse(err)
	}

	defer resp.Body.Close()

	r := newRPMResponse(nil).AddStatusCode(resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	if r.GetError() == nil {
		r.SetError(err)
	}
	r.AddBody(body)
	return r
}

func ProcessData(data []interface{}, runId string) []interface{} {
	if len(data) > 0 {
		data[0] = runId
	}
	return data
}

func NewAPMClient(conf *config.Configuration, FunctionName string) (RpmCmd, *RpmControls, error) {
	cmd := RpmCmd{
		Collector: conf.NewRelicHost,
	}

	cs := RpmControls{
		License: conf.LicenseKey,
		Client: &http.Client{
			Timeout: 1000 * time.Second,
		},
		GzipWriterPool: &sync.Pool{
			New: func() interface{} {
				w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
				return w
			},
		},
		FunctionName: FunctionName,
	}
	if conf.PreconnectEnabled {
		// Perform Preconnect before sending cmd
		redirectHost, err := apmPreConnect(cmd, &cs)
		if err != nil {
			return RpmCmd{}, nil, fmt.Errorf("preconnect failed: %w", err)
		}
		cmd.Collector = redirectHost
	}
	_, _, err := apmConnect(cmd, &cs)
	if err != nil {
		return RpmCmd{}, nil, fmt.Errorf("connect failed: %w", err)
	}
	// Close the channel to signal that the connect process is done
	close(ConnectDone)

	return cmd, &cs, nil
}

func apmPreConnect(apmCmd RpmCmd, apmControls *RpmControls) (string, error) {
	// PRE-CONNECT
	apmCmd.Name = cmdPreconnect
	startTime := time.Now()
	redirectHost, err := PreConnect(apmCmd, apmControls)
	if err != nil {
		return "", fmt.Errorf("pre-connect failed: %w", err)
	}
	duration := time.Since(startTime)
	fmt.Printf("Redirect Host: %s\n", redirectHost)
	fmt.Printf("Pre-Connect Cycle duration: %s\n", duration)
	return redirectHost, nil
}

func apmConnect(apmCmd RpmCmd, apmControls *RpmControls) (string, string, error) {
	// CONNECT
	apmCmd.Name = cmdConnect
	startTime := time.Now()
	runID, entityGUID, err := Connect(apmCmd, apmControls)
	if err != nil {
		return "", "", fmt.Errorf("connect failed: %w", err)
	}
	duration := time.Since(startTime)
	fmt.Printf("Run ID: %s\n", runID)
	fmt.Printf("Entity GUID: %s\n", entityGUID)
	fmt.Printf("Connect Cycle duration: %s\n", duration)
	return runID, entityGUID, nil
}