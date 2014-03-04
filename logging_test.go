package logging_test

import (
	"../logging"
	"testing"
)

func TestOutput(t *testing.T) {
	logging.L.SetLogFile("")
	logging.L.Info(map[string]string{"key 1": "value 1", "key2": "value2"}, "Hello World %s\t{%d}", "An\targument", 1234)
}

// test basic logging functionality
func TestLogger(t *testing.T) {
	err := logging.L.SetLogFile("/dev/null")
	if err != nil {
		t.Errorf("Could not set log file: %s", err)
	}
	// turn off Fatal logging so that we don't crash the process
	logging.L.SetOutput(true, true, true, true, false)
	err = logging.L.Error(nil, "Hello")
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	err = logging.L.Info(nil, "Hello %s", "World")
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	err = logging.L.Debug(nil, "Hello %s", "World", "Extra")
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	err = logging.L.Critical(nil, "Hello %s\t%d", "World", 1234)
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	err = logging.L.Error(nil, "{Hello %s\t%d}", "World", 1234)
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	err = logging.L.Fatal(map[string]string{"key 1": "value 1", "key2": "value2"}, "")
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
}

// benchmark logging calls that write to /dev/null
func BenchmarkNullLogger(b *testing.B) {
	logging.L.SetLogFile("/dev/null")
	for i := 0; i < b.N; i++ {
		logging.L.Info(map[string]string{"key 1": "value 1", "key2": "value2"}, "Hello World %s\t{%d}", "An\targument", 1234)
	}
}

// benchmark logging calls that write to a file
func BenchmarkFileLogger(b *testing.B) {
	logging.L.SetLogFile("./benchmark.log")
	for i := 0; i < b.N; i++ {
		logging.L.Info(map[string]string{"key 1": "value 1", "key2": "value2"}, "Hello World %s\t{%d}", "An\targument", 1234)
	}
}

// benchmark logging calls that don't actually do anything; tests map setup
func BenchmarkStubbedLogger(b *testing.B) {
	logging.L.SetOutput(false, false, false, false, false)
	for i := 0; i < b.N; i++ {
		logging.L.Info(map[string]string{"key 1": "value 1", "key2": "value2"}, "Hello World %s\t{%d}", "An\targument", 1234)
	}
}
