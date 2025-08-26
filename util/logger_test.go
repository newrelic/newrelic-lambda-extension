package util

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLogger(t *testing.T) {
    tests := []struct {
        name         string
        logsEnabled  bool
        logLevel     string
        expectDebug  bool
        expectOutput bool
    }{
        {"Logs enabled with INFO level", true, "INFO", false, true},
        {"Logs enabled with DEBUG level", true, "DEBUG", true, true},
        {"Logs disabled", false, "INFO", false, false},
        {"Case insensitive debug", true, "debug", true, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            log.SetOutput(&buf)
            defer log.SetOutput(os.Stderr)

            ConfigLogger(tt.logsEnabled, tt.logLevel)

            assert.Equal(t, tt.logsEnabled, logger.isEnabled)
            assert.Equal(t, strings.ToUpper(tt.logLevel), logger.logLevel)
            assert.Equal(t, tt.expectDebug, logger.isDebugEnabled)

            if tt.expectOutput {
                assert.Contains(t, buf.String(), "New Relic Lambda Extension starting up")
            } else {
                assert.Empty(t, buf.String())
            }
        })
    }
}

func TestDebugf(t *testing.T) {
    tests := []struct {
        name        string
        logsEnabled bool
        logLevel    string
        expectLog   bool
    }{
        {"Debug enabled with debug level", true, "DEBUG", true},
        {"Debug disabled with info level", true, "INFO", false},
        {"Logs disabled", false, "DEBUG", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            log.SetOutput(&buf)
            defer log.SetOutput(os.Stderr)

            ConfigLogger(tt.logsEnabled, tt.logLevel)
            buf.Reset()

            Debugf("Test %s", "message")

            if tt.expectLog {
                assert.Contains(t, buf.String(), "Test message")
                assert.Contains(t, buf.String(), "[NR_EXT DEBUG]")
            } else {
                assert.Empty(t, buf.String())
            }
        })
    }
}

func TestDebugln(t *testing.T) {
    tests := []struct {
        name        string
        logsEnabled bool
        logLevel    string
        expectLog   bool
    }{
        {"Debug enabled with debug level", true, "DEBUG", true},
        {"Debug disabled with info level", true, "INFO", false},
        {"Logs disabled", false, "DEBUG", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            log.SetOutput(&buf)
            defer log.SetOutput(os.Stderr)

            ConfigLogger(tt.logsEnabled, tt.logLevel)
            buf.Reset()

            Debugln("Test", "message")

            if tt.expectLog {
                assert.Contains(t, buf.String(), "Test message")
                assert.Contains(t, buf.String(), "[NR_EXT DEBUG]")
            } else {
                assert.Empty(t, buf.String())
            }
        })
    }
}

func TestLogf(t *testing.T) {
    tests := []struct {
        name        string
        logsEnabled bool
        expectLog   bool
    }{
        {"Logs enabled", true, true},
        {"Logs disabled", false, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            log.SetOutput(&buf)
            defer log.SetOutput(os.Stderr)

            ConfigLogger(tt.logsEnabled, "INFO")
            buf.Reset()

            Logf("Info %s", "message")

            if tt.expectLog {
                assert.Contains(t, buf.String(), "Info message")
                assert.Contains(t, buf.String(), "[NR_EXT INFO]")
            } else {
                assert.Empty(t, buf.String())
            }
        })
    }
}

func TestLogln(t *testing.T) {
    tests := []struct {
        name        string
        logsEnabled bool
        expectLog   bool
    }{
        {"Logs enabled", true, true},
        {"Logs disabled", false, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            log.SetOutput(&buf)
            defer log.SetOutput(os.Stderr)

            ConfigLogger(tt.logsEnabled, "INFO")
            buf.Reset()

            Logln("Info", "message")

            if tt.expectLog {
                assert.Contains(t, buf.String(), "Info message")
                assert.Contains(t, buf.String(), "[NR_EXT INFO]")
            } else {
                assert.Empty(t, buf.String())
            }
        })
    }
}

func TestLoggerMethods(t *testing.T) {
    var buf bytes.Buffer
    log.SetOutput(&buf)
    defer log.SetOutput(os.Stderr)

    ConfigLogger(true, "DEBUG")
    buf.Reset()

    testLogger := Logger{isEnabled: true, isDebugEnabled: true, logLevel: "DEBUG"}

    testLogger.Debugf("Debug %s", "test")
    assert.Contains(t, buf.String(), "Debug test")
    buf.Reset()

    testLogger.Debugln("Debug", "line")
    assert.Contains(t, buf.String(), "Debug line")
    buf.Reset()

    testLogger.Logf("Log %s", "test")
    assert.Contains(t, buf.String(), "Log test")
    buf.Reset()

    testLogger.Logln("Log", "line")
    assert.Contains(t, buf.String(), "Log line")
}

func TestLogLevelConstants(t *testing.T) {
    assert.Equal(t, "INFO", LogLevelInfo)
    assert.Equal(t, "DEBUG", LogLevelDebug)
}
