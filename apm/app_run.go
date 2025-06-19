package apm

import (
	"sync"
)

type appRun struct {
	Reply       *ConnectReply
	firstAppame string
	mu           sync.RWMutex
}

func newAppRun(config apmConfig, reply *ConnectReply) *appRun {
	return &appRun{
		Reply: reply,
	}
}
