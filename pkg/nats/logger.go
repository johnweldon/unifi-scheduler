package nats

import (
	"io"
	"log"

	"github.com/nats-io/nats.go"
)

type Logger struct {
	Connection     *nats.Conn
	LogFlags       int
	LogPrefix      string
	PublishSubject string
}

func NewStdLogger(l *Logger) *log.Logger { return log.New(l, l.LogPrefix, l.LogFlags) }

func (l *Logger) Write(p []byte) (n int, err error) {
	if err := l.Connection.Publish(l.PublishSubject, p); err != nil {
		return -1, err
	}

	return len(p), nil
}

var _ io.Writer = (*Logger)(nil)
