package telemetry

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/newrelic/lambda-extension/util"
)

const (
	maxCompressedSizeBytes = 1 << 20
)

// request contains an http.Request and the UncompressedBody which is provided
// for logging.
type request struct {
	Request          *http.Request
	UncompressedBody json.RawMessage

	compressedBodyLength int
}

type requestsBuilder interface {
	makeBody() json.RawMessage
	split() []requestsBuilder
}

var (
	errUnableToSplit = fmt.Errorf("unable to split large payload further")
)

func requestNeedsSplit(r request) bool {
	return r.compressedBodyLength >= maxCompressedSizeBytes
}

func newRequests(batch requestsBuilder, licenseKey string, url string, userAgent string) ([]request, error) {
	return newRequestsInternal(batch, licenseKey, url, userAgent, requestNeedsSplit)
}

func newRequestsInternal(batch requestsBuilder, licenseKey string, url string, userAgent string, needsSplit func(request) bool) ([]request, error) {
	uncompressed := batch.makeBody()

	compressed, err := util.Compress(uncompressed)
	if err != nil {
		return nil, fmt.Errorf("error compressing data: %v", err)
	}

	compressedLen := compressed.Len()

	req, err := http.NewRequest("POST", url, compressed)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("X-License-Key", licenseKey)

	r := request{
		Request:              req,
		UncompressedBody:     uncompressed,
		compressedBodyLength: compressedLen,
	}

	if !needsSplit(r) {
		return []request{r}, nil
	}

	batches := batch.split()
	if batches == nil {
		return nil, errUnableToSplit
	}

	var reqs []request

	for _, b := range batches {
		rs, err := newRequestsInternal(b, licenseKey, url, userAgent, needsSplit)
		if err != nil {
			return nil, err
		}

		reqs = append(reqs, rs...)
	}

	return reqs, nil
}
