package util

import "log"

var logger = Logger{
	isDebugEnabled: false,
}

type Logger struct {
	isDebugEnabled bool
}

func ConfigLogger(isDebugEnabled bool) {
	// Go Logging config
	log.SetPrefix("[NR_EXT] ")
	log.SetFlags(0)

	log.Println("New Relic Lambda Extension starting up")

	logger.isDebugEnabled = isDebugEnabled
}

func (l Logger) Debugf(format string, v ...interface{}) {
	if l.isDebugEnabled {
		log.Printf(format, v...)
	}
}

func (l Logger) Debugln(v ...interface{}) {
	if l.isDebugEnabled {
		log.Println(v...)
	}
}

func (l Logger) Logf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l Logger) Logln(v ...interface{}) {
	log.Println(v...)
}

func Debugf(format string, v ...interface{}) {
	if logger.isDebugEnabled {
		log.Printf(format, v...)
	}
}

func Debugln(v ...interface{}) {
	if logger.isDebugEnabled {
		log.Println(v...)
	}
}

func Logf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func Logln(v ...interface{}) {
	log.Println(v...)
}

func Fatal(v ...interface{}) {
	log.Fatal(v...)
}

func Panic(v ...interface{}) {
	log.Panic(v...)
}
