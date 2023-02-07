package util

import "time"

const (
	Name    = "newrelic-lambda-extension"
	Version = "3.0.0"
	Id      = Name + ":" + Version
)

var (
	//TODO: make this much lower once collector repaired
	SendToNewRelicTimeout = 2400 * time.Millisecond
)
