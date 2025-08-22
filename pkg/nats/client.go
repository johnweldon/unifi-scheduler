// Package nats provides NATS JetStream integration for the unifi-scheduler.
//
// This package implements a distributed caching and messaging system using NATS JetStream
// to store and distribute UniFi controller data across multiple instances of the scheduler.
// It supports both publish/subscribe messaging and key-value storage for efficient data
// sharing and real-time updates.
//
// Key components:
//   - Client: Core NATS connection and JetStream management
//   - Publisher: High-level interface for publishing data and events
//   - Subscriber: High-level interface for consuming cached data
//   - Agent: Long-running service that populates the cache from UniFi controllers
//
// The package provides automatic connection management, stream creation, and key-value
// bucket setup with configurable replication and retention policies.
//
// Basic usage:
//
//	// Create a publisher for caching data
//	publisher := nats.NewPublisher(
//	    nats.OptNATSUrl("nats://server:4222"),
//	    nats.OptStreamReplicas(3),
//	)
//
//	// Store client data in the cache
//	err := publisher.Store("clients", "active", clientList)
//
//	// Create a subscriber for reading cached data
//	subscriber := nats.NewSubscriber(
//	    nats.OptNATSUrl("nats://server:4222"),
//	)
//
//	// Retrieve cached client data
//	var clients []unifi.Client
//	err = subscriber.Get("clients", "active", &clients)
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

// Default configuration values for NATS operations.
// These provide reasonable defaults for most UniFi controller scenarios.
var (
	// DefaultConnectTimeout is the default timeout for establishing NATS connections
	DefaultConnectTimeout = 15 * time.Second

	// DefaultWriteTimeout is the default timeout for write operations to NATS
	DefaultWriteTimeout = 30 * time.Second

	// DefaultOperationTimeout is the default timeout for general NATS operations
	DefaultOperationTimeout = 30 * time.Second

	// DefaultStreamReplicas is the default replication factor for JetStream streams
	DefaultStreamReplicas = 3

	// DefaultKVReplicas is the default replication factor for key-value buckets
	DefaultKVReplicas = 3
)

// ClientOpt represents a configuration option for NATS clients.
//
// Options follow the functional options pattern, allowing flexible configuration
// of NATS clients without requiring large constructor parameter lists.
type ClientOpt func(*Client)

// OptNATSUrl configures the NATS server URL for client connections.
//
// The URL should include the protocol and port (e.g., "nats://server:4222").
// For TLS connections, use "tls://server:4222". For WebSocket connections,
// use "ws://server:8080" or "wss://server:443".
func OptNATSUrl(u string) ClientOpt { return func(c *Client) { c.connURL = u } }

// OptCreds configures the path to a NATS credentials file for authentication.
//
// The credentials file should contain the JWT and NKey for authenticating
// with a secured NATS server. This is the recommended authentication method
// for production deployments.
func OptCreds(credsFilePath string) ClientOpt { return func(c *Client) { c.credsFile = credsFilePath } }

// OptStreams configures the JetStream streams that the client will manage.
//
// Streams are used for persistent messaging and event distribution. The client
// will automatically create these streams if they don't exist, with appropriate
// configuration for UniFi controller data.
func OptStreams(names ...string) ClientOpt {
	return func(c *Client) { c.streams = append(c.streams, names...) }
}

func OptBuckets(names ...string) ClientOpt {
	return func(c *Client) { c.buckets = append(c.buckets, names...) }
}

func OptConnectTimeout(t time.Duration) ClientOpt {
	return func(c *Client) { c.connectTimeout = t }
}

func OptWriteTimeout(t time.Duration) ClientOpt {
	return func(c *Client) { c.writeTimeout = t }
}

func OptOperationTimeout(t time.Duration) ClientOpt {
	return func(c *Client) { c.operationTimeout = t }
}

func OptStreamReplicas(r int) ClientOpt {
	return func(c *Client) { c.streamReplicas = r }
}

func OptKVReplicas(r int) ClientOpt {
	return func(c *Client) { c.kvReplicas = r }
}

type Client struct {
	connURL   string
	credsFile string
	conn      *nats.Conn
	streams   []string
	buckets   []string

	connectTimeout   time.Duration
	writeTimeout     time.Duration
	operationTimeout time.Duration
	streamReplicas   int
	kvReplicas       int
}

func (n *Client) Init(opts ...ClientOpt) {
	// Set defaults
	n.connectTimeout = DefaultConnectTimeout
	n.writeTimeout = DefaultWriteTimeout
	n.operationTimeout = DefaultOperationTimeout
	n.streamReplicas = DefaultStreamReplicas
	n.kvReplicas = DefaultKVReplicas

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
		nats.Timeout(n.connectTimeout),
		nats.FlusherTimeout(n.writeTimeout),
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
	return n.ensureStreamsWithContext(context.Background())
}

func (n *Client) ensureStreamsWithContext(ctx context.Context) error {
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
			Replicas:   n.streamReplicas,
		}

		opCtx, cancel := context.WithTimeout(ctx, n.operationTimeout)
		defer cancel()

		if _, err = js.Stream(opCtx, stream); err != nil {
			if !errors.Is(err, jetstream.ErrStreamNotFound) {
				return fmt.Errorf("ensureStreams: getting stream info %q: %w", stream, err)
			}

			if _, err = js.CreateStream(opCtx, cfg); err != nil {
				return fmt.Errorf("ensureStreams: creating stream %q: %w", stream, err)
			}
		}

		if _, err = js.UpdateStream(opCtx, cfg); err != nil {
			return fmt.Errorf("ensureStreams: updating stream %q: %w", stream, err)
		}
	}

	return nil
}

func (n *Client) ensureBuckets() error {
	return n.ensureBucketsWithContext(context.Background())
}

func (n *Client) ensureBucketsWithContext(ctx context.Context) error {
	var (
		err error
		js  jetstream.JetStream
	)

	if js, err = jetstream.New(n.conn); err != nil {
		return fmt.Errorf("ensureBuckets: cannot get jetstream: %w", err)
	}

	for _, bucket := range n.buckets {
		opCtx, cancel := context.WithTimeout(ctx, n.operationTimeout)
		defer cancel()

		if _, err = js.KeyValue(opCtx, bucket); err != nil {
			if !errors.Is(err, jetstream.ErrBucketNotFound) {
				return fmt.Errorf("ensureBuckets: getting bucket %q: %w", bucket, err)
			}

			cfg := jetstream.KeyValueConfig{
				Bucket:   bucket,
				TTL:      90 * 24 * time.Hour,
				Replicas: n.kvReplicas,
			}

			if _, err = js.CreateKeyValue(opCtx, cfg); err != nil {
				return fmt.Errorf("ensureBuckets: creating bucket %q: %w", bucket, err)
			}
		}
	}

	return nil
}
