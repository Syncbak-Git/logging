package logging

import (
	"encoding/json"
	"fmt"
)

type Logger interface {
	SetLogFile(fileName string)
	Error(values map[string]string, message ...string)
}

type logger struct {
}

func (l logger) Error(values map[string]string, message ...string) {
	l.writeEntry("ERROR", values, message)
}

func (l logger) writeEntry(severity string, values map[string]string, message ...string) {
	kv := l.getHeaderValues()
	headerStr := makeHeaderString(kv)
	messageStr := fmt.Sprintf(message...)
	jsonStr := makeJsonString(kv, values, messageStr)
	logEntry := fmt.Sprintf("%s\t%s\t%s\n", headerStr, messageStr, jsonStr)
	l.write(logEntry)
}
