package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
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

// Level is a logging level. Multiple logging levels can be combined by ORing individual
// Level values, eg. Debug|Error will log both DEBUG and ERROR entries.
type Level uint64

const (
	Debug Level = 1 << iota
	Info
	Warning
	Error
	Critical
	Fatal
	Metrics
	All  = 0xFFFF
	None = 0
)

// New creates a new private Logger. If fileName is an empty string, the Logger will write to stdout. New will return nil if it can't create/open fileName.
func New(fileName string) *Logger {
	hostname, _ := os.Hostname()
	l := Logger{
		appName:  path.Base(os.Args[0]),
		hostName: hostname,
		pid:      strconv.Itoa(os.Getpid()),
		logLevel: All,
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
	logLevel    Level
}

// SetLogFile sets fileName as the log file target. An empty string sets text file logging to stdout
// and json logging to null (ie, no json output); this is the default.
// Normally, there are two log files, one for text and one for json. The text file will be written to
// filename, while the json content will be written to filename.json
// If filename.json cannot be opened for write (eg, filename = "/dev/null"), then
// both text and json will be written to filename.
func (l *Logger) SetLogFile(fileName string) error {
	// TODO: change this so writes to the json file always go through a channel. When the
	// channel is nil, we can skip the preparation of the json. When we want to write
	// to a file, launch a goroutine that listens to the channel and writes to the file.
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

// SetOutput controls which levels of logging are enabled/disabled. Obsolete -- use SetLogLevel()
func (l *Logger) SetOutput(debug, info, warning, err, critical, fatal bool) {
	l.logLevel = None
	if debug {
		l.logLevel |= Debug
	}
	if info {
		l.logLevel |= Info
	}
	if err {
		l.logLevel |= Error
	}
	if warning {
		l.logLevel |= Warning
	}
	if fatal {
		l.logLevel |= Fatal
	}
	if critical {
		l.logLevel |= Critical
	}
}

// EnableAllOutput is the same as SetLogLevel(All) and is
// a convenience function that may be useful for easily enabling logging during tests.
func (l *Logger) EnableAllOutput() {
	l.logLevel = All
}

// SetLogLevel sets the logging level to the level or levels in level, eg Debug | Error
func (l *Logger) SetLogLevel(level Level) {
	l.logLevel = level
}

// Error is like Debug for ERROR log entries.
func (l *Logger) Error(values map[string]interface{}, format string, args ...interface{}) error {
	if l.logLevel&Error == 0 {
		return nil
	}
	return l.writeEntry(Error, values, format, args...)
}

// Warning is like Debug for WARNING log entries.
func (l *Logger) Warning(values map[string]interface{}, format string, args ...interface{}) error {
	if l.logLevel&Warning == 0 {
		return nil
	}
	return l.writeEntry(Warning, values, format, args...)
}

// Info is like Debug for INFO log entries.
func (l *Logger) Info(values map[string]interface{}, format string, args ...interface{}) error {
	if l.logLevel&Info == 0 {
		return nil
	}
	return l.writeEntry(Info, values, format, args...)
}

// Debug writes a DEBUG log entry. The optional values map contains
// user-supplied key-value pairs. format and args are passed to fmt.Printf
// to generate the message entry.
func (l *Logger) Debug(values map[string]interface{}, format string, args ...interface{}) error {
	if l.logLevel&Debug == 0 {
		return nil
	}
	return l.writeEntry(Debug, values, format, args...)
}

// Metrics is like Debug for METRICS log entries.
func (l *Logger) Metrics(values map[string]interface{}, format string, args ...interface{}) error {
	if l.logLevel&Metrics == 0 {
		return nil
	}
	return l.writeEntry(Metrics, values, format, args...)
}

// Critical is like Debug for CRITICAL log entries.
func (l *Logger) Critical(values map[string]interface{}, format string, args ...interface{}) error {
	if l.logLevel&Critical == 0 {
		return nil
	}
	return l.writeEntry(Critical, values, format, args...)
}

// Fatal is like Debug for FATAL log entries, but it also calls os.Exit(1).
func (l *Logger) Fatal(values map[string]interface{}, format string, args ...interface{}) error {
	if l.logLevel&Fatal == 0 {
		return nil
	}
	err := l.writeEntry(Fatal, values, format, args...)
	os.Exit(1)
	return err // won't actually get here
}

func (l Level) _String(logLevel Level) string {
	s := make([]string, 0)
	if l&Debug != 0 && logLevel&Debug != 0 {
		s = append(s, "DEBUG")
	}
	if l&Info != 0 && logLevel&Info != 0 {
		s = append(s, "INFO")
	}
	if l&Warning != 0 && logLevel&Warning != 0 {
		s = append(s, "WARNING")
	}
	if l&Error != 0 && logLevel&Error != 0 {
		s = append(s, "ERROR")
	}
	if l&Critical != 0 && logLevel&Critical != 0 {
		s = append(s, "CRITICAL")
	}
	if l&Fatal != 0 && logLevel&Fatal != 0 {
		s = append(s, "FATAL")
	}
	if l&Metrics != 0 && logLevel&Metrics != 0 {
		s = append(s, "METRICS")
	}
	return strings.Join(s, "|")
}

// Write allows writing log entries with multiple log levels, eg DEBUG|INFO|METRICS
func (l *Logger) Write(level Level, values map[string]interface{}, format string, args ...interface{}) error {
	err := l.writeEntry(level, values, format, args...)
	if level&Fatal != 0 {
		os.Exit(1)
	}
	return err
}

// NewKV is a convenience function that creates a map of (string) keys and
// associated values that can be passed as the values argument to Debug(), Info(), etc.
// args is a set of key + value pairs.
func NewKV(args ...interface{}) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		s, ok := args[i].(string)
		if !ok {
			return nil, fmt.Errorf("args[%d] is not a string: %+v", i, args[i])
		}
		var a interface{}
		if i+1 < len(args) {
			a = args[i+1]
		}
		m[s] = a
	}
	return m, nil
}

var re *regexp.Regexp = regexp.MustCompile("ERROR|FATAL|CRITICAL")

func (l *Logger) writeEntry(level Level, values map[string]interface{}, format string, args ...interface{}) error {
	if level&l.logLevel == 0 {
		return nil
	}
	kv := l.getHeaderValues(level)
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
	//only write to json file/channel on certain levels or when we have a map
	if values != nil || re.MatchString(headerStr) {
		jsonStr, err := makeJSONString(kv, values, messageStr)
		if err != nil {
			return err
		}
		if l.jsonChannel != nil {
			l.jsonChannel <- jsonStr
		} else {
			_, err = fmt.Fprintln(l.jsonWriter, jsonStr)
		}
	}
	return err
}

func (l *Logger) getHeaderValues(level Level) map[string]interface{} {
	pc, file, line, _ := runtime.Caller(3)
	f := runtime.FuncForPC(pc)
	caller := f.Name()
	m := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"severity":  level._String(l.logLevel),
		"pid":       l.pid,
		"app":       l.appName,
		"host":      l.hostName,
		"line":      strconv.Itoa(line),
		"file":      path.Base(file),
		"function":  path.Base(caller),
	}
	return m
}

func makeHeaderString(m map[string]interface{}) string {
	return strings.Join([]string{m["timestamp"].(string), m["severity"].(string)}, "\t")
}

func makeJSONString(header map[string]interface{}, kv map[string]interface{}, message string) (string, error) {
	merged := make(map[string]interface{})
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
