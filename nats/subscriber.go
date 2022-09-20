package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func NewSubscriber(opts ...ClientOpt) *Subscriber {
	p := &Subscriber{}
	p.Init(opts...)
	return p
}

type Subscriber struct {
	Client
}

func (s *Subscriber) Get(bucket, key string, into any) error { return s.retrieve(bucket, key, into) }

func (s *Subscriber) Subscribe(ctx context.Context, subjects ...string) (<-chan string, error) {
	return s.subscribe(ctx, subjects...)
}

func (s *Subscriber) SubscribeStream(ctx context.Context, subjects ...string) (<-chan string, error) {
	return s.subscribeStream(ctx, subjects...)
}

func (s *Subscriber) retrieve(bucket, key string, into any) error {
	var (
		err   error
		js    nats.JetStreamContext
		kv    nats.KeyValue
		entry nats.KeyValueEntry
	)

	if err = s.ensureConnection(); err != nil {
		return fmt.Errorf("retrieve: not connected: %w", err)
	}

	if js, err = s.conn.JetStream(); err != nil {
		return fmt.Errorf("retrieve: cannot get jetstream: %w", err)
	}

	if kv, err = js.KeyValue(bucket); err != nil {
		return fmt.Errorf("retrieve: cannot get bucket %q: %w", bucket, err)
	}

	if entry, err = kv.Get(key); err != nil {
		return fmt.Errorf("retrieve: cannot get %q in bucket %q: %w", key, bucket, err)
	}

	if err = json.Unmarshal(entry.Value(), into); err != nil {
		return fmt.Errorf("retrieve: cannot unmarshal %q from bucket %q: %w", key, bucket, err)
	}

	return nil
}

func (s *Subscriber) subscribe(ctx context.Context, subjects ...string) (<-chan string, error) {
	var err error

	streams := s.streams
	if len(subjects) > 0 {
		streams = subjects
	}

	if len(streams) < 1 {
		return nil, fmt.Errorf("subscribe: no stream names")
	}

	if err = s.ensureConnection(); err != nil {
		return nil, fmt.Errorf("subscribe: not connected: %w", err)
	}

	var subscriptions []*nats.Subscription

	for _, stream := range streams {
		var ss *nats.Subscription
		if ss, err = s.conn.SubscribeSync(stream); err != nil {
			break
		}

		subscriptions = append(subscriptions, ss)
	}

	if err != nil {
		for _, ss := range subscriptions {
			if e2 := ss.Unsubscribe(); err != nil && !errors.Is(e2, nats.ErrBadSubscription) {
				err = fmt.Errorf("while rolling back subscriptions because of %w, got: %v", err, e2)
			}
		}

		return nil, fmt.Errorf("subscribe: cannot subscribe: %w", err)
	}

	evt := make(chan string)

	for _, sub := range subscriptions {
		go func(ctx context.Context, ss *nats.Subscription, evt chan<- string) {
			defer func() { _ = ss.Unsubscribe() }()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					msg, err := ss.NextMsg(time.Second)
					if err != nil {
						if errors.Is(err, nats.ErrTimeout) {
							continue
						}

						log.Printf("in subscription loop: %v", err)
						return
					}

					if msg.Data != nil {
						evt <- string(msg.Data)
					}
				}
			}
		}(ctx, sub, evt)
	}

	return evt, nil
}

func (s *Subscriber) subscribeStream(ctx context.Context, subjects ...string) (<-chan string, error) {
	var (
		err error
		js  nats.JetStreamContext
	)

	streams := s.streams
	if len(subjects) > 0 {
		streams = subjects
	}

	if len(streams) < 1 {
		return nil, fmt.Errorf("subscribe: no stream names")
	}

	if err = s.ensureConnection(); err != nil {
		return nil, fmt.Errorf("subscribe: not connected: %w", err)
	}

	if js, err = s.conn.JetStream(); err != nil {
		return nil, fmt.Errorf("subscribe: cannot get jetstream: %w", err)
	}

	var subscriptions []*nats.Subscription

	for _, stream := range streams {
		var ss *nats.Subscription
		if ss, err = js.SubscribeSync(stream); err != nil {
			break
		}

		subscriptions = append(subscriptions, ss)
	}

	if err != nil {
		for _, ss := range subscriptions {
			if e2 := ss.Unsubscribe(); err != nil && !errors.Is(e2, nats.ErrBadSubscription) {
				err = fmt.Errorf("while rolling back subscriptions because of %w, got: %v", err, e2)
			}
		}

		return nil, fmt.Errorf("subscribe: cannot subscribe: %w", err)
	}

	evt := make(chan string)

	for _, sub := range subscriptions {
		go func(ctx context.Context, ss *nats.Subscription, evt chan<- string) {
			defer func() { _ = ss.Unsubscribe() }()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					msg, err := ss.NextMsg(time.Second)
					if err != nil {
						if errors.Is(err, nats.ErrTimeout) {
							continue
						}

						log.Printf("in subscription loop: %v", err)
						return
					}

					if msg.Data != nil {
						evt <- string(msg.Data)
					}

					if err = msg.InProgress(); err != nil {
						log.Printf("marking msg in progress: %v", err)
					}
				}
			}
		}(ctx, sub, evt)
	}

	return evt, nil
}
