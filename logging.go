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

// L is the global Logger instance.
var L *Logger

func init() {
	L = New("")
}

// New creates a new private Logger. If fileName is an empty string, the Logger will write to stdout. New will return nil if it can't create/open fileName.
func New(fileName string) *Logger {
	hostname, _ := os.Hostname()
	l := Logger{
		appName:    path.Base(os.Args[0]),
		hostName:   hostname,
		pid:        strconv.Itoa(os.Getpid()),
		doDebug:    false,
		doInfo:     true,
		doWarning:  true,
		doError:    true,
		doCritical: true,
		doFatal:    true,
	}
	err := l.SetLogFile(fileName)
	if err != nil {
		return nil
	}
	return &l
}

// Logger is a type passed to the logging functions. It stores the log settings.
type Logger struct {
	appName     string
	hostName    string
	pid         string
	jsonWriter  io.WriteCloser
	textWriter  io.WriteCloser
	jsonChannel chan<- string
	doDebug     bool
	doInfo      bool
	doWarning   bool
	doError     bool
	doCritical  bool
	doFatal     bool
}

// SetLogFile sets fileName as the log file target. An empty string sets text file logging to stdout
// and json logging to null (ie, no json output); this is the default.
// Normally, there are two log files, one for text and one for json. The text file will be written to
// filename, while the json content will be written to filename.json
// If filename.json cannot be opened for write (eg, filename = "/dev/null"), then
// both text and json will be written to filename.
func (l *Logger) SetLogFile(fileName string) error {
	var textWriter, jsonWriter *os.File
	var err error
	if len(fileName) == 0 {
		textWriter = os.Stdout
		jsonWriter, _ = os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	} else {
		textWriter, err = os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err == nil {
			jsonWriter, err = os.OpenFile(fmt.Sprintf("%s.json", fileName), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
			if err != nil {
				jsonWriter = textWriter
				err = nil
			}
		}
	}
	if err == nil {
		if l.textWriter != nil && l.textWriter != os.Stdout {
			l.textWriter.Close()
		}
		if l.jsonWriter != nil && l.jsonWriter != os.Stdout && l.jsonWriter != l.textWriter {
			l.jsonWriter.Close()
		}
		l.textWriter = textWriter
		l.jsonWriter = jsonWriter
	}
	return err
}

// WriteJSONToChannel changes the destination of the json entries from a local file to a channel. Note that
// SetLogFile will change it back to using a local file, so keep the call order in mind.
func (l *Logger) WriteJSONToChannel(c chan<- string) {
	l.jsonChannel = c
}

// SetOutput controls which levels of logging are enabled/disabled
func (l *Logger) SetOutput(debug, info, warning, err, critical, fatal bool) {
	l.doDebug = debug
	l.doInfo = info
	l.doWarning = warning
	l.doError = err
	l.doCritical = critical
	l.doFatal = fatal
}

// Error is like Debug for ERROR log entries.
func (l *Logger) Error(values map[string]string, format string, args ...interface{}) error {
	if !l.doError {
		return nil
	}
	return l.writeEntry("ERROR", values, format, args...)
}

// Warning is like Debug for WARNING log entries.
func (l *Logger) Warning(values map[string]string, format string, args ...interface{}) error {
	if !l.doWarning {
		return nil
	}
	return l.writeEntry("WARNING", values, format, args...)
}

// Info is like Debug for INFO log entries.
func (l *Logger) Info(values map[string]string, format string, args ...interface{}) error {
	if !l.doInfo {
		return nil
	}
	return l.writeEntry("INFO", values, format, args...)
}

// Debug writes a DEBUG log entry. The optional values map contains
// user-supplied key-value pairs. format and args are passed to fmt.Printf
// to generate the message entry.
func (l *Logger) Debug(values map[string]string, format string, args ...interface{}) error {
	if !l.doDebug {
		return nil
	}
	return l.writeEntry("DEBUG", values, format, args...)
}

// Critical is like Debug for CRITICAL log entries.
func (l *Logger) Critical(values map[string]string, format string, args ...interface{}) error {
	if !l.doCritical {
		return nil
	}
	return l.writeEntry("CRITICAL", values, format, args...)
}

// Fatal is like Debug for FATAL log entries, but it also calls os.Exit(1).
func (l *Logger) Fatal(values map[string]string, format string, args ...interface{}) error {
	if !l.doFatal {
		return nil
	}
	err := l.writeEntry("FATAL", values, format, args...)
	os.Exit(1)
	return err // won't actually get here
}

func (l *Logger) writeEntry(severity string, values map[string]string, format string, args ...interface{}) error {
	kv := l.getHeaderValues(severity)
	headerStr := makeHeaderString(kv)
	messageStr := fmt.Sprintf(format, args...)
	if strings.ContainsAny(messageStr, "{}\t") {
		messageStr = strings.Replace(messageStr, "\t", " ", -1)
		messageStr = strings.Replace(messageStr, "{", "[", -1)
		messageStr = strings.Replace(messageStr, "}", "]", -1)
	}
	_, err := fmt.Fprintf(l.textWriter, "%s\t%s\n", headerStr, messageStr)
	if err != nil {
		return err
	}
	jsonStr, err := makeJSONString(kv, values, messageStr)
	if err != nil {
		return err
	}
	if l.jsonChannel != nil {
		l.jsonChannel <- jsonStr
	} else {
		_, err = fmt.Fprintln(l.jsonWriter, jsonStr)
	}
	return err
}

func (l *Logger) getHeaderValues(severity string) map[string]string {
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

func makeJSONString(header map[string]string, kv map[string]string, message string) (string, error) {
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
