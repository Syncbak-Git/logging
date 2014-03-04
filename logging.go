// Package logging provides basic logging with machine-readable output.
package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Logger provides functions for setting up logging and writing log entries.
type Logger interface {
	SetLogFile(fileName string) error
	SetOutput(debug, info, err, critical, fatal bool)
	Debug(values map[string]string, format string, args ...interface{}) error
	Info(values map[string]string, format string, args ...interface{}) error
	Error(values map[string]string, format string, args ...interface{}) error
	Critical(values map[string]string, format string, args ...interface{}) error
	Fatal(values map[string]string, format string, args ...interface{}) error
}

// L is the global logger instance
var L logger

// New creates a new private Logger. If fileName is an empty string, the Logger will write to stdout. New will return nil if it can't create/open fileName.
func New(fileName string) Logger {
	hostname, _ := os.Hostname()
	l = logger{
		appName:    path.Base(os.Args[0]),
		hostName:   hostname,
		pid:        strconv.Itoa(os.Getpid()),
		doDebug:    false,
		doInfo:     true,
		doError:    true,
		doCritical: true,
		doFatal:    true,
	}
	err := l.SetLogFile(fileName)
	if err != nil {
		return nil
	}
	return l
}

type logger struct {
	appName    string
	hostName   string
	pid        string
	writer     io.WriteCloser
	doDebug    bool
	doInfo     bool
	doError    bool
	doCritical bool
	doFatal    bool
}

func init() {
	L = New("")
}

// SetLogFile sets fileName as the log file target. An empty string sets logging to stdout (the default).
func (l *logger) SetLogFile(fileName string) error {
	var w *os.File
	var err error
	if len(fileName) == 0 {
		w = os.Stdout
	} else {
		w, err = os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	}
	if err == nil {
		if l.writer != nil && l.writer != os.Stdout {
			l.writer.Close()
		}
		l.writer = w
	}
	return err
}

// SetOutput controls which levels of logging are enabled/disabled
func (l *logger) SetOutput(debug, info, err, critical, fatal bool) {
	l.doDebug = debug
	l.doInfo = info
	l.doError = err
	l.doCritical = critical
	l.doFatal = fatal
}

// Error is like Debug for ERROR log entries.
func (l *logger) Error(values map[string]string, format string, args ...interface{}) error {
	if !l.doError {
		return nil
	}
	return l.writeEntry("ERROR", values, format, args...)
}

// Info is like Debug for INFO log entries.
func (l *logger) Info(values map[string]string, format string, args ...interface{}) error {
	if !l.doInfo {
		return nil
	}
	return l.writeEntry("INFO", values, format, args...)
}

// Debug writes a DEBUG log entry. The optional values map contains
// user-supplied key-value pairs. format and args are passed to fmt.Printf
// to generate the message entry.
func (l *logger) Debug(values map[string]string, format string, args ...interface{}) error {
	if !l.doDebug {
		return nil
	}
	return l.writeEntry("DEBUG", values, format, args...)
}

// Critical is like Debug for CRITICAL log entries.
func (l *logger) Critical(values map[string]string, format string, args ...interface{}) error {
	if !l.doCritical {
		return nil
	}
	return l.writeEntry("CRITICAL", values, format, args...)
}

// Fatal is like Debug for FATAL log entries, but it also calls os.Exit(1).
func (l *logger) Fatal(values map[string]string, format string, args ...interface{}) error {
	if !l.doFatal {
		return nil
	}
	err := l.writeEntry("FATAL", values, format, args...)
	os.Exit(1)
	return err // won't actually get here
}

func (l *logger) writeEntry(severity string, values map[string]string, format string, args ...interface{}) error {
	kv := l.getHeaderValues(severity)
	headerStr := makeHeaderString(kv)
	messageStr := fmt.Sprintf(format, args...)
	if strings.ContainsAny(messageStr, "{}\t") {
		messageStr = strings.Replace(messageStr, "\t", " ", -1)
		messageStr = strings.Replace(messageStr, "{", "[", -1)
		messageStr = strings.Replace(messageStr, "}", "]", -1)
	}
	jsonStr, err := makeJsonString(kv, values, messageStr)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(l.writer, "%s\t%s\t%s\n", headerStr, messageStr, jsonStr)
	return err
}

func (l *logger) getHeaderValues(severity string) map[string]string {
	pc, file, line, _ := runtime.Caller(3)
	f := runtime.FuncForPC(pc)
	caller := f.Name()
	m := map[string]string{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"severity":  severity,
		"pid":       l.pid,
		"app":       l.appName,
		"host":      l.hostName,
		"line":      strconv.Itoa(line),
		"file":      path.Base(file),
		"function":  path.Base(caller),
	}
	return m
}

func makeHeaderString(m map[string]string) string {
	return strings.Join([]string{m["timestamp"], m["severity"]}, "\t")
}

func makeJsonString(header map[string]string, kv map[string]string, message string) (string, error) {
	merged := make(map[string]string)
	for k, v := range kv {
		merged[k] = v
	}
	for k, v := range header {
		merged[k] = v
	}
	merged["message"] = message
	b, err := json.Marshal(merged)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
