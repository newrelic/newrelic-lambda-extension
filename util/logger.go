package util

import "log"

var logger = Logger{
	isEnabled:      true,
	isDebugEnabled: false,
}

type Logger struct {
	isEnabled      bool
	isDebugEnabled bool
}

func ConfigLogger(logsEnabled bool, isDebugEnabled bool) {
	// Go Logging config
	log.SetPrefix("[NR_EXT] ")
	log.SetFlags(0)

	log.Println("New Relic Lambda Extension starting up")

	logger.isEnabled = logsEnabled
	logger.isDebugEnabled = isDebugEnabled
}

func (l Logger) Debugf(format string, v ...interface{}) {
	if l.isEnabled && l.isDebugEnabled {
		log.Printf(format, v...)
	}
}

func (l Logger) Debugln(v ...interface{}) {
	if l.isEnabled && l.isDebugEnabled {
		log.Println(v...)
	}
}

func (l Logger) Infof(format string, v ...interface{}) {
	if l.isEnabled {
		log.Printf(format, v...)
	}
}

func (l Logger) Infoln(v ...interface{}) {
	if l.isEnabled {
		log.Println(v...)
	}
}

func Debugf(format string, v ...interface{}) {
	if logger.isEnabled && logger.isDebugEnabled {
		log.Printf(format, v...)
	}
}

func Debugln(v ...interface{}) {
	if logger.isEnabled && logger.isDebugEnabled {
		log.Println(v...)
	}
}

func Infof(format string, v ...interface{}) {
	if logger.isEnabled {
		log.Printf(format, v...)
	}
}

func Infoln(v ...interface{}) {
	if logger.isEnabled {
		log.Println(v...)
	}
}

func Fatal(v ...interface{}) {
	if logger.isEnabled {
		log.Fatal(v...)
	}
}

func Panic(v ...interface{}) {
	if logger.isEnabled {
		log.Panic(v...)
	}
}
