package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func NewPublisher(opts ...ClientOpt) *Publisher {
	p := &Publisher{}
	p.Init(opts...)
	return p
}

type Publisher struct {
	Client
}

func (n *Publisher) Store(bucket, key string, val any) error {
	return n.StoreWithContext(context.Background(), bucket, key, val)
}

func (n *Publisher) StoreWithContext(ctx context.Context, bucket, key string, val any) error {
	return n.storeWithContext(ctx, bucket, key, val)
}

func (n *Publisher) Publish(subject string, msg any) error { return n.publish(subject, msg) }

func (n *Publisher) PublishStream(stream, subject string, msg any) error {
	return n.PublishStreamWithContext(context.Background(), stream, subject, msg)
}

func (n *Publisher) PublishStreamWithContext(ctx context.Context, stream, subject string, msg any) error {
	return n.publishStreamWithContext(ctx, fmt.Sprintf("%s.%s", stream, subject), msg)
}

func (n *Publisher) publish(subject string, msg any) error {
	var (
		err  error
		data []byte
	)

	if err = n.ensureConnection(); err != nil {
		return fmt.Errorf("publish: not connected: %w", err)
	}

	if data, err = json.Marshal(msg); err != nil {
		return fmt.Errorf("publish: cannot marshal data: %w", err)
	}

	if err = n.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish: cannot publish data: %w", err)
	}

	return nil
}

func (n *Publisher) publishStream(stream string, msg any) error {
	return n.publishStreamWithContext(context.Background(), stream, msg)
}

func (n *Publisher) publishStreamWithContext(ctx context.Context, stream string, msg any) error {
	var (
		err  error
		data []byte
		js   jetstream.JetStream
	)

	if err = n.ensureConnection(); err != nil {
		return fmt.Errorf("publishStream: not connected: %w", err)
	}

	if js, err = jetstream.New(n.conn); err != nil {
		return fmt.Errorf("publishStream: cannot get jetstream: %w", err)
	}

	if data, err = json.Marshal(msg); err != nil {
		return fmt.Errorf("publishStream: cannot marshal data: %w", err)
	}

	hdr := nats.Header{}
	if nid, ok := msg.(natsIDProvider); ok {
		hdr.Set("Nats-Msg-Id", nid.UniqueID())
	}

	pmsg := &nats.Msg{
		Subject: stream,
		Header:  hdr,
		Data:    data,
	}

	opCtx, cancel := context.WithTimeout(ctx, n.operationTimeout)
	defer cancel()

	if _, err = js.PublishMsg(opCtx, pmsg); err != nil {
		return fmt.Errorf("publishStream: cannot publish data: %w", err)
	}

	return nil
}

func (n *Publisher) store(bucket, key string, val any) error {
	return n.storeWithContext(context.Background(), bucket, key, val)
}

func (n *Publisher) storeWithContext(ctx context.Context, bucket, key string, val any) error {
	var (
		err  error
		data []byte
		js   jetstream.JetStream
		kv   jetstream.KeyValue
	)

	if err = n.ensureConnection(); err != nil {
		return fmt.Errorf("store: not connected: %w", err)
	}

	if js, err = jetstream.New(n.conn); err != nil {
		return fmt.Errorf("store: cannot get jetstream: %w", err)
	}

	opCtx, cancel := context.WithTimeout(ctx, n.operationTimeout)
	defer cancel()

	if kv, err = js.KeyValue(opCtx, bucket); err != nil {
		return fmt.Errorf("store: cannot get bucket %q: %w", bucket, err)
	}

	if data, err = json.Marshal(val); err != nil {
		return fmt.Errorf("store: cannot marshal data: %w", err)
	}

	if _, err = kv.Put(opCtx, key, data); err != nil {
		return fmt.Errorf("store: cannot put in bucket %q: %w", bucket, err)
	}

	return nil
}

type natsIDProvider interface {
	UniqueID() string
}
