package telemetryApi

import (
	"bytes"
)

const (
	// LogLevelFieldName is the name of the log level field in New Relic logging JSON
	LogLevelFieldName = "level"

	// LogMessageFieldName is the name of the log message field in New Relic logging JSON
	LogMessageFieldName = "message"

	// LogTimestampFieldName is the name of the timestamp field in New Relic logging JSON
	LogTimestampFieldName = "timestamp"

	// LogSpanIDFieldName is the name of the span ID field in the New Relic logging JSON
	LogSpanIDFieldName = "span.id"

	// LogTraceIDFieldName is the name of the trace ID field in the New Relic logging JSON
	LogTraceIDFieldName = "trace.id"

	// LogSeverityUnknown is the value the log severity should be set to if no log severity is known
	LogSeverityUnknown = "UNKNOWN"

	// JSON Attribute Constants
	HostnameAttributeKey   = "hostname"
	EntityNameAttributeKey = "entity.name"
	entityGUIDAttributeKey = "entity.guid"

	MaxLogLength = 32768
)

type LogPayload struct {
	*bytes.Buffer
	done bool
}

// NewLogLine creates an object for processing a single log line and sending it to New Relic
func NewLogPayload(commonAttributes map[string]string) *LogPayload {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte('[')
	buf.WriteByte('{')
	buf.WriteString(`"common":`)
	buf.WriteByte('{')
	buf.WriteString(`"attributes":`)
	buf.WriteByte('{')

	for name, value := range commonAttributes {
		name = "\"" + name + "\":"
		buf.WriteString(name)
		AppendString(buf, value)
		buf.WriteByte(',')
	}
	buf.WriteByte('}')
	buf.WriteByte('}')
	buf.WriteByte(',')
	buf.WriteString(`"logs":`)
	buf.WriteByte('[')

	return &LogPayload{Buffer: buf}
}

// AddLogLine prepares a Log Event JSON object in the format expected by the collector.
// Timestamp must be unix millisecond time
func (buf *LogPayload) AddLogLine(Timestamp int64, Level, Message string) {
	if buf.done {
		return
	}

	if Level == "" {
		Level = LogSeverityUnknown
	}

	if len(Message) > MaxLogLength {
		Message = Message[:MaxLogLength]
	}

	w := jsonFieldsWriter{buf: buf.Buffer}
	buf.WriteByte('{')
	w.stringField(LogLevelFieldName, Level)
	w.stringField(LogMessageFieldName, Message)

	w.needsComma = false
	buf.WriteByte(',')
	w.intField(LogTimestampFieldName, Timestamp)
	buf.WriteByte('}')
}

func (buf *LogPayload) Marshal() []byte {
	if buf.done {
		return buf.Bytes()
	}

	// prevent Duplication of JSON closure
	buf.done = true

	buf.WriteByte(']')
	buf.WriteByte('}')
	buf.WriteByte(']')
	return buf.Bytes()
}
