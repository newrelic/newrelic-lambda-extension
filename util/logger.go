package util

import (
	"fmt"
	"log"
	"strings"
)

const (
    LogLevelInfo  = "INFO"
    LogLevelDebug = "DEBUG"
)

var logger = Logger{
    isEnabled:      true,
    isDebugEnabled: false,
    logLevel:       LogLevelInfo,
}

type Logger struct {
    isEnabled      bool
    isDebugEnabled bool
    logLevel       string
}

func ConfigLogger(logsEnabled bool, logLevel string) {
    // Go Logging config
    log.SetPrefix("")
    log.SetFlags(0)

    logger.isEnabled = logsEnabled
    logger.logLevel = strings.ToUpper(logLevel)
    logger.isDebugEnabled = (logger.logLevel == LogLevelDebug)

    if logger.isEnabled {
        log.Printf("[NR_EXT %s] New Relic Lambda Extension starting up\n", logger.logLevel)
    }
}

func (l Logger) Debugf(format string, v ...interface{}) {
    if l.isEnabled && l.isDebugEnabled {
        log.Printf("[NR_EXT %s] "+format, append([]interface{}{LogLevelDebug}, v...)...)
    }
}

func (l Logger) Debugln(v ...interface{}) {
    if l.isEnabled && l.isDebugEnabled {
        log.Printf("[NR_EXT %s] %s\n", LogLevelDebug, fmt.Sprint(v...))
    }
}

func (l Logger) Logf(format string, v ...interface{}) {
    if l.isEnabled {
        log.Printf("[NR_EXT %s] "+format, append([]interface{}{LogLevelInfo}, v...)...)
    }
}

func (l Logger) Logln(v ...interface{}) {
    if l.isEnabled {
        log.Printf("[NR_EXT %s] %s\n", LogLevelInfo, fmt.Sprint(v...))
    }
}

func Debugf(format string, v ...interface{}) {
    if logger.isEnabled && logger.isDebugEnabled {
        log.Printf("[NR_EXT %s] "+format, append([]interface{}{LogLevelDebug}, v...)...)
    }
}

func Debugln(v ...interface{}) {
    if logger.isEnabled && logger.isDebugEnabled {
        log.Printf("[NR_EXT %s] %s\n", LogLevelDebug, fmt.Sprint(v...))
    }
}

func Logf(format string, v ...interface{}) {
    if logger.isEnabled {
        log.Printf("[NR_EXT %s] "+format, append([]interface{}{LogLevelInfo}, v...)...)
    }
}

func Logln(v ...interface{}) {
    if logger.isEnabled {
        log.Printf("[NR_EXT %s] %s\n", LogLevelInfo, fmt.Sprint(v...))
    }
}

func Fatal(v ...interface{}) {
    log.Fatalf("[NR_EXT ERROR] %s", fmt.Sprint(v...))
}

func Panic(v ...interface{}) {
    log.Panicf("[NR_EXT ERROR] %s\n", fmt.Sprint(v...))
}
