package checks

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockClientError struct{}

func (c *mockClientError) Get(string) (*http.Response, error) {
	return nil, errors.New("Something went wrong")
}

type mockClientRedirect struct{}

func (c *mockClientRedirect) Get(string) (*http.Response, error) {
	body := ioutil.NopCloser(bytes.NewBufferString("Hello World"))
	return &http.Response{Body: body, StatusCode: 301}, nil
}

func TestRuntimeCheck(t *testing.T) {
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	oldPath := runtimeLookupPath
	defer func() {
		runtimeLookupPath = oldPath
	}()
	runtimeLookupPath = filepath.Join(dirname, runtimeLookupPath)

	os.MkdirAll(filepath.Join(runtimeLookupPath, "node"), os.ModePerm)
	r, err := checkAndReturnRuntime()
	assert.Equal(t, runtimeConfigs[Node].language, r.language)
	assert.Nil(t, err)
}

func TestRuntimeCheckNil(t *testing.T) {
	r, err := checkAndReturnRuntime()
	assert.Equal(t, runtimeConfig{}, r)
	assert.Nil(t, err)
}

func TestLatestAgentTag(t *testing.T) {
	client = &mockClientError{}
	assert.Nil(t, latestAgentTag(&runtimeConfig{}))

	client = &mockClientRedirect{}
	assert.Nil(t, latestAgentTag(&runtimeConfig{}))
}
