package nats

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

func NewPublisher(opts ...ClientOpt) *Publisher {
	p := &Publisher{}
	p.Init(opts...)
	return p
}

type Publisher struct {
	Client
}

func (n *Publisher) Store(bucket, key string, val any) error { return n.store(bucket, key, val) }

func (n *Publisher) Publish(subject string, msg any) error { return n.publish(subject, msg) }

func (n *Publisher) PublishStream(subject string, msg any) error {
	return n.publishStream(subject, msg)
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
	var (
		err  error
		data []byte
		js   nats.JetStreamContext
	)

	if err = n.ensureConnection(); err != nil {
		return fmt.Errorf("publishStream: not connected: %w", err)
	}

	if js, err = n.conn.JetStream(); err != nil {
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

	if _, err = js.PublishMsgAsync(pmsg); err != nil {
		return fmt.Errorf("publishStream: cannot publish data: %w", err)
	}

	return nil
}

func (n *Publisher) store(bucket, key string, val any) error {
	var (
		err  error
		data []byte
		js   nats.JetStreamContext
		kv   nats.KeyValue
	)

	if err = n.ensureConnection(); err != nil {
		return fmt.Errorf("store: not connected: %w", err)
	}

	if js, err = n.conn.JetStream(); err != nil {
		return fmt.Errorf("store: cannot get jetstream: %w", err)
	}

	if kv, err = js.KeyValue(bucket); err != nil {
		return fmt.Errorf("store: cannot get bucket %q: %w", bucket, err)
	}

	if data, err = json.Marshal(val); err != nil {
		return fmt.Errorf("store: cannot marshal data: %w", err)
	}

	if _, err = kv.Put(key, data); err != nil {
		return fmt.Errorf("store: cannot get put in bucket %q: %w", bucket, err)
	}

	return nil
}

type natsIDProvider interface {
	UniqueID() string
}
