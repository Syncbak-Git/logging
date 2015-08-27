package logging_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Syncbak-Git/logging"
)

func Example() {
	l := logging.New("")
	val, _ := logging.NewKV("key 1", "value 1", "key2", "value2")
	l.Info(val, "Hello World %s\t{%d}", "An\targument", 1234)
	// Expected output (adjust timestamps):
	// 2014-03-04T21:48:45.925788398Z  INFO    Hello World An argument [1234]
	// Json file output, if we had any:
	// {"app":"logging.test","file":"logging_test.go","function":"logging_test.TestOutput",
	// "host":"kenf-linux","key 1":"value 1","key2":"value2","line":"10",
	// "message":"Hello World An argument [1234]","pid":"5992","severity":"INFO",
	// "timestamp":"2014-03-04T21:48:45.925788398Z"}
}

func ExampleNewKV() {
	val, _ := logging.NewKV("Hello", "world", "number", 1234)
	fmt.Println(val)
	// Output: map[Hello:world number:1234]
}

// test basic logging functionality
func testLogger(f string, usePitcher bool, t *testing.T) {
	err := logging.L.SetLogFile(f)
	if err != nil {
		t.Errorf("Could not set log file: %s", err)
	}
	interval := time.Duration(2) * time.Second
	if usePitcher {
		jsonChannel := make(chan string)
		defer close(jsonChannel)
		go func() {
			for {
				select {
				case s, isOpen := <-jsonChannel:
					if isOpen {
						fmt.Printf("JSON Channel: %s\n", s)
					} else {
						return
					}
				}
			}
		}()
		logging.L.WriteJSONToChannel(jsonChannel)
	}
	// turn off Fatal logging so that we don't crash the process
	logging.L.SetOutput(true, true, true, true, true, false)
	err = logging.L.Debug(nil, "Hello %s", "World", "Extra")
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	err = logging.L.Info(nil, "Hello %s", "World")
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	err = logging.L.Error(nil, "Hello")
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	err = logging.L.Warning(nil, "{Hello %s\t%d}", "World", 1234)
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	err = logging.L.Critical(nil, "Hello %s\t%d", "World", 1234)
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	err = logging.L.Fatal(map[string]interface{}{"key 1": "value 1", "key2": "value2"}, "")
	if err != nil {
		t.Errorf("Log write error: %s", err)
	}
	if usePitcher {
		time.Sleep(interval * 2)
	}
}

func TestLogger(t *testing.T) {
	fileNames := []string{
		"",
		"/dev/null",
		"./testlog.log",
	}
	defer os.Remove("./testlog.log")
	defer os.Remove("./testlog.log.json")
	for _, f := range fileNames {
		testLogger(f, false, t)
	}
}

func TestPitcher(t *testing.T) {
	fileNames := []string{
		"",
		"/dev/null",
		"./testlog.log",
	}
	defer os.Remove("./testlog.log")
	for _, f := range fileNames {
		testLogger(f, true, t)
	}
}

// benchmark logging calls that write to /dev/null
func BenchmarkNullLogger(b *testing.B) {
	logging.L.SetLogFile("/dev/null")
	for i := 0; i < b.N; i++ {
		logging.L.Info(map[string]interface{}{"key 1": "value 1", "key2": "value2"}, "Hello World %s\t{%d}", "An\targument", 1234)
	}
}

// benchmark logging calls that write to a file
func BenchmarkFileLogger(b *testing.B) {
	logging.L.SetLogFile("./benchmark.log")
	defer os.Remove("./benchmark.log")
	defer os.Remove("./benchmark.log.json")
	for i := 0; i < b.N; i++ {
		logging.L.Info(map[string]interface{}{"key 1": "value 1", "key2": "value2"}, "Hello World %s\t{%d}", "An\targument", 1234)
	}
}

// benchmark logging calls that don't actually do anything; tests map setup
func BenchmarkStubbedLogger(b *testing.B) {
	logging.L.SetLogLevel(logging.None)
	for i := 0; i < b.N; i++ {
		logging.L.Info(map[string]interface{}{"key 1": "value 1", "key2": "value2"}, "Hello World %s\t{%d}", "An\targument", 1234)
	}
}

func TestLogger_Write(t *testing.T) {
	l := logging.New("./writetest.log")
	if l == nil {
		t.Fatal("Could not create writetest.log")
	}
	defer os.Remove("./writetest.log")
	defer os.Remove("./writetest.log.json")
	l.SetLogLevel(logging.Error)
	err := l.Write(logging.Error|logging.Debug, nil, "Hello world")
	human, err := ioutil.ReadFile("./writetest.log")
	if err != nil {
		t.Fatalf("Could not read log file: %s", err)
	}
	if strings.Count(string(human), "ERROR") != 1 {
		t.Errorf("Bad ERROR count:\n%s", string(human))
	}
	if strings.Count(string(human), "DEBUG") != 0 {
		t.Errorf("Bad DEBUG count:\n%s", string(human))
	}
	js, err := ioutil.ReadFile("./writetest.log.json")
	if err != nil {
		t.Fatalf("Could not read json log file: %s", err)
	}
	if strings.Count(string(js), "ERROR") != 1 {
		t.Errorf("Bad ERROR count:\n%s", string(human))
	}
	if strings.Count(string(js), "DEBUG") != 0 {
		t.Errorf("Bad DEBUG count:\n%s", string(human))
	}
}
