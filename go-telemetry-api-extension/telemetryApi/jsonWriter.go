package telemetryApi

import (
	"bytes"
)

type jsonWriter interface {
	WriteJSON(buf *bytes.Buffer)
}

type jsonFieldsWriter struct {
	buf        *bytes.Buffer
	needsComma bool
}

func (w *jsonFieldsWriter) addKey(key string) {
	if w.needsComma {
		w.buf.WriteByte(',')
	} else {
		w.needsComma = true
	}
	// defensively assume that the key needs escaping:
	AppendString(w.buf, key)
	w.buf.WriteByte(':')
}

func (w *jsonFieldsWriter) stringField(key string, val string) {
	w.addKey(key)
	AppendString(w.buf, val)
}

func (w *jsonFieldsWriter) intField(key string, val int64) {
	w.addKey(key)
	AppendInt(w.buf, val)
}

func (w *jsonFieldsWriter) floatField(key string, val float64) {
	w.addKey(key)
	AppendFloat(w.buf, val)
}

func (w *jsonFieldsWriter) float32Field(key string, val float32) {
	w.addKey(key)
	AppendFloat32(w.buf, val)
}

func (w *jsonFieldsWriter) boolField(key string, val bool) {
	w.addKey(key)
	if val {
		w.buf.WriteString("true")
	} else {
		w.buf.WriteString("false")
	}
}

func (w *jsonFieldsWriter) rawField(key string, val jsonString) {
	w.addKey(key)
	w.buf.WriteString(string(val))
}

func (w *jsonFieldsWriter) writerField(key string, val jsonWriter) {
	w.addKey(key)
	val.WriteJSON(w.buf)
}
