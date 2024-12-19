package nats

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	DefaultConnectTimeout = 15 * time.Second
	DefaultWriteTimeout   = 30 * time.Second
)

type ClientOpt func(*Client)

func OptNATSUrl(u string) ClientOpt { return func(c *Client) { c.connURL = u } }

func OptCreds(credsFilePath string) ClientOpt { return func(c *Client) { c.credsFile = credsFilePath } }

func OptStreams(names ...string) ClientOpt {
	return func(c *Client) { c.streams = append(c.streams, names...) }
}

func OptBuckets(names ...string) ClientOpt {
	return func(c *Client) { c.buckets = append(c.buckets, names...) }
}

type Client struct {
	connURL   string
	credsFile string
	conn      *nats.Conn
	streams   []string
	buckets   []string
}

func (n *Client) Init(opts ...ClientOpt) {
	for _, opt := range opts {
		opt(n)
	}
}

func (n *Client) ensureConnection() error {
	var err error

	if n.conn != nil {
		if n.conn.IsConnected() {
			return nil
		}

		log.Print("ensureConnection: was not connected, retrying")

		n.conn.Close()
		n.conn = nil
	}

	opts := []nats.Option{
		nats.Timeout(DefaultConnectTimeout),
		nats.FlusherTimeout(DefaultWriteTimeout),
	}

	if len(n.credsFile) != 0 {
		opts = append(opts, nats.UserCredentials(n.credsFile))
	}

	if n.conn, err = nats.Connect(n.connURL, opts...); err != nil {
		return fmt.Errorf("ensureConnection: connecting to NATS: %w", err)
	}

	if err = n.ensureStreams(); err != nil {
		return fmt.Errorf("ensureConnection: ensuring streams:  %w", err)
	}

	if err = n.ensureBuckets(); err != nil {
		return fmt.Errorf("ensureConnection: ensuring buckets:  %w", err)
	}

	return nil
}

func (n *Client) ensureStreams() error {
	var (
		err error
		js  jetstream.JetStream
	)

	if js, err = jetstream.New(n.conn); err != nil {
		return fmt.Errorf("ensureStreams: cannot get jetstream: %w", err)
	}

	for _, stream := range n.streams {
		cfg := jetstream.StreamConfig{
			Name:       stream,
			Subjects:   []string{fmt.Sprintf("%s.*", stream)},
			Duplicates: 1 * time.Hour,
			Discard:    jetstream.DiscardOld,
			Retention:  jetstream.LimitsPolicy,
			MaxMsgs:    1000,
			Replicas:   3,
		}

		if _, err = js.Stream(context.Background(), stream); err != nil {
			if !errors.Is(err, jetstream.ErrStreamNotFound) {
				return fmt.Errorf("ensureStreams: getting stream info %q: %w", stream, err)
			}

			if _, err = js.CreateStream(context.Background(), cfg); err != nil {
				return fmt.Errorf("ensureStreams: creating stream %q: %w", stream, err)
			}
		}

		if _, err = js.UpdateStream(context.Background(), cfg); err != nil {
			return fmt.Errorf("ensureStreams: updating stream %q: %w", stream, err)
		}
	}

	return nil
}

func (n *Client) ensureBuckets() error {
	var (
		err error
		js  jetstream.JetStream
	)

	if js, err = jetstream.New(n.conn); err != nil {
		return fmt.Errorf("ensureBuckets: cannot get jetstream: %w", err)
	}

	for _, bucket := range n.buckets {
		if _, err = js.KeyValue(context.Background(), bucket); err != nil {
			if !errors.Is(err, jetstream.ErrBucketNotFound) {
				return fmt.Errorf("ensureBuckets: getting bucket %q: %w", bucket, err)
			}

			cfg := jetstream.KeyValueConfig{
				Bucket:   bucket,
				TTL:      90 * 24 * time.Hour,
				Replicas: 3,
			}

			if _, err = js.CreateKeyValue(context.Background(), cfg); err != nil {
				return fmt.Errorf("ensureBuckets: creating bucket %q: %w", bucket, err)
			}
		}
	}

	return nil
}
