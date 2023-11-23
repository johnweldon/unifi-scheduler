package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
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
		js    jetstream.JetStream
		kv    jetstream.KeyValue
		entry jetstream.KeyValueEntry
	)

	if err = s.ensureConnection(); err != nil {
		return fmt.Errorf("retrieve: not connected: %w", err)
	}

	if js, err = jetstream.New(s.conn); err != nil {
		return fmt.Errorf("retrieve: cannot get jetstream: %w", err)
	}

	if kv, err = js.KeyValue(context.Background(), bucket); err != nil {
		return fmt.Errorf("retrieve: cannot get bucket %q: %w", bucket, err)
	}

	if entry, err = kv.Get(context.Background(), key); err != nil {
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
		js  jetstream.JetStream
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

	if js, err = jetstream.New(s.conn); err != nil {
		return nil, fmt.Errorf("subscribe: cannot get jetstream: %w", err)
	}

	var consumers []jetstream.Consumer

	cfg := jetstream.ConsumerConfig{
		InactiveThreshold: 100 * time.Millisecond,
	}

	for _, stream := range streams {
		var cons jetstream.Consumer
		if cons, err = js.CreateOrUpdateConsumer(context.Background(), stream, cfg); err != nil {
			return nil, fmt.Errorf("subscribe: cannot create consumer: %w", err)
		}

		consumers = append(consumers, cons)
	}

	evt := make(chan string)

	for _, cons := range consumers {
		go func(ctx context.Context, cons jetstream.Consumer, evt chan<- string) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					msg, err := cons.Next()
					if err != nil {
						if errors.Is(err, nats.ErrTimeout) {
							continue
						}

						log.Printf("in subscription loop: %v", err)
						return
					}

					if data := msg.Data(); data != nil {
						evt <- string(data)
					}

					if err = msg.InProgress(); err != nil {
						log.Printf("marking msg in progress: %v", err)
					}
				}
			}
		}(context.Background(), cons, evt)
	}

	return evt, nil
}
