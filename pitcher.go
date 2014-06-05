package logging

import (
	"encoding/gob"
	"fmt"
	"github.com/Syncbak-Git/sbupload/encryption"
	"io"
	"net"
	"time"
)

type LogTransfer struct {
	Source    string
	StartTime time.Time
	Lines     []string
}

type transferConfig struct {
	host              string
	port              string
	password          string
	reportingInterval time.Duration
	source            string
	entry             chan []byte
	quit              chan interface{}
	log               *Logger
}

func newCatcherWriter(host, port, password string, interval time.Duration, source string, l *Logger) io.WriteCloser {
	t := transferConfig{
		host:              host,
		port:              port,
		password:          password,
		reportingInterval: interval,
		source:            source,
		entry:             make(chan []byte),
		quit:              make(chan interface{}),
		log:               l,
	}
	go transferLogEntries(t)
	return &t
}

func (t *transferConfig) Close() error {
	close(t.quit)
	return nil
}

func (t *transferConfig) Write(p []byte) (n int, err error) {
	if p == nil || len(p) == 0 {
		return 0, nil
	}
	t.entry <- p
	return len(p), nil
}

func connect(host string, port string, password string) (*gob.Encoder, *gob.Decoder, error) {
	addr := net.JoinHostPort(host, port)
	addresses, err := net.LookupHost(host)
	if err != nil || addresses == nil || len(addresses) == 0 {
		return nil, nil, fmt.Errorf("DNS lookup failed for %s: %s", host, err)
	} else {
		addr = net.JoinHostPort(addresses[0], port)
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not connect to %s: %s", addr, err)
	}
	encoder := gob.NewEncoder(conn)
	token, err := encryption.MakeAuthToken(password, "logupload1")
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("Error creating auth token to %s: %s", addr, err)
	}
	protocol := int32(1)
	err = encoder.Encode(protocol)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("Error sending protocol to %s: %s", addr, err)
	}
	err = encoder.Encode(token)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("Error sending auth token to %s: %s", addr, err)
	}
	return encoder, gob.NewDecoder(conn), nil
}

func transferLogEntries(settings transferConfig) {
	var enc *gob.Encoder
	var err error
	lines := make([]string, 0, 200)
	timer := time.Tick(settings.reportingInterval)
	for {
		select {
		case <-timer:
			if len(lines) > 0 {
				if enc == nil {
					enc, _, err = connect(settings.host, settings.port, settings.password)
					if err != nil {
						settings.log.writeLocalEntry("ERROR", nil, "Could not connect to %s:, %s", settings.host, err)
					}
				}
				if enc != nil {
					t := LogTransfer{
						Source:    settings.source,
						StartTime: time.Now().Add(-settings.reportingInterval),
						Lines:     lines,
					}
					err = enc.Encode(t)
					if err != nil {
						settings.log.writeLocalEntry("ERROR", nil, "Error sending to %s:, %s", settings.host, err)
						enc = nil
					} else {
						lines = make([]string, 0, cap(lines))
					}
				}
			}
		case s := <-settings.entry:
			lines = append(lines, string(s))
		case <-settings.quit:
			return
		}
	}
}
