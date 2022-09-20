package nats

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/johnweldon/unifi-scheduler/unifi"
)

func NewAgent(s *unifi.Session, base string, opts ...ClientOpt) *Agent {
	addnl := []ClientOpt{
		OptBuckets(detailBucket(base), byMACBucket(base), byNameBucket(base)),
		OptStreams(eventStream(base)),
	}

	return &Agent{
		client:    s,
		publisher: NewPublisher(append(opts, addnl...)...),
		base:      base,
	}
}

type Agent struct {
	client    *unifi.Session
	publisher *Publisher
	base      string
}

func (a *Agent) Start(ctx context.Context) error {
	if a.client == nil {
		return errors.New("missing unifi client")
	}

	if a.publisher == nil {
		return errors.New("missing publisher")
	}

	if a.base == "" {
		return errors.New("missing base name")
	}

	go a.serve(ctx)

	return nil
}

func (a *Agent) serve(ctx context.Context) {
	var err error

	eventInterval := time.After(1 * time.Second)
	lookupInterval := time.After(5 * time.Second)
	clientInterval := time.After(10 * time.Second)
	userInterval := time.After(20 * time.Second)
	deviceInterval := time.After(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return

		case <-eventInterval:
			eventInterval = time.After(10 * time.Second)
			if err = a.publishEvents(); err != nil {
				log.Println(err)
			}

		case <-clientInterval:
			clientInterval = time.After(1 * time.Minute)
			if err = a.refreshClients(); err != nil {
				log.Println(err)
			}

		case <-userInterval:
			userInterval = time.After(5 * time.Minute)
			if err = a.refreshUsers(); err != nil {
				log.Println(err)
			}

		case <-deviceInterval:
			deviceInterval = time.After(10 * time.Minute)
			if err = a.refreshDevices(); err != nil {
				log.Println(err)
			}

		case <-lookupInterval:
			lookupInterval = time.After(15 * time.Minute)
			if err = a.refreshLookups(); err != nil {
				log.Println(err)
			}
		}
	}
}

func (a *Agent) publishEvents() error {
	events, err := a.client.GetRecentEvents()
	if err != nil {
		return fmt.Errorf("get events: %w", err)
	}

	for _, evt := range events {
		if err = a.publishStream(eventStream(a.base), evt); err != nil {
			return fmt.Errorf("publish events: %w", err)
		}
	}

	return nil
}

func (a *Agent) refreshClients() error {
	clients, err := a.client.GetClients()
	if err != nil {
		return fmt.Errorf("get clients: %w", err)
	}

	if err = a.publish("clients", clients); err != nil {
		return err
	}

	if err = a.store(detailBucket(a.base), "active", clients); err != nil {
		return fmt.Errorf("persisting live clients: %w", err)
	}

	return nil
}

func (a *Agent) refreshUsers() error {
	users, err := a.client.GetUsers()
	if err != nil {
		return fmt.Errorf("get users: %w", err)
	}

	if err = a.publish("users", users); err != nil {
		return err
	}

	for _, user := range users {
		mac := string(user.MAC)
		if err = a.store(detailBucket(a.base), mac, user); err != nil {
			return fmt.Errorf("persisting user %q: %w", mac, err)
		}
	}

	return nil
}

func (a *Agent) refreshDevices() error {
	devices, err := a.client.GetDevices()
	if err != nil {
		return fmt.Errorf("get devices: %w", err)
	}

	if err = a.publish("devices", devices); err != nil {
		return err
	}

	for _, device := range devices {
		mac := string(device.MAC)
		if err = a.store(detailBucket(a.base), mac, device); err != nil {
			return fmt.Errorf("persisting device %q: %w", mac, err)
		}
	}

	return nil
}

func (a *Agent) refreshLookups() error {
	macs, err := a.client.GetMACs()
	if err != nil {
		return fmt.Errorf("get MACs: %w", err)
	}

	for k, v := range macs {
		mac := string(k)
		if err = a.store(byMACBucket(a.base), mac, v); err != nil {
			return fmt.Errorf("persisting MAC names %q: %w", mac, err)
		}
	}

	names, err := a.client.GetNames()
	if err != nil {
		return fmt.Errorf("get names: %w", err)
	}

	for k, v := range names {
		if err = a.store(byNameBucket(a.base), k, v); err != nil {
			return fmt.Errorf("persisting name MACs %q: %w", k, err)
		}
	}

	return nil
}

func (a *Agent) publish(subject string, msg any) error {
	if err := a.publisher.Publish(subSubject(a.base, subject), msg); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

func (a *Agent) publishStream(stream string, msg any) error {
	if err := a.publisher.PublishStream(stream, msg); err != nil {
		return fmt.Errorf("publish stream %q: %w", stream, err)
	}

	return nil
}

func (a *Agent) store(bucket, key string, val any) error {
	norm := normalize(key)
	if norm == "" {
		return nil
	}

	if err := a.publisher.Store(bucket, norm, val); err != nil {
		return fmt.Errorf("storing key %q: %w", norm, err)
	}

	return nil
}

func detailBucket(base string) string        { return base + "-details" }
func byMACBucket(base string) string         { return base + "-bymac" }
func byNameBucket(base string) string        { return base + "-byname" }
func eventStream(base string) string         { return base + "-events" }
func subSubject(base, subject string) string { return strings.Join([]string{base, subject}, ".") }

func normalize(s string) string {
	fn := func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '.':
			return r
		case r == '-':
			return r
		case r == ':':
			return '-'
		case r == ' ':
			return '-'
		default:
			return -1
		}
	}

	return strings.Map(fn, strings.ToLower(strings.TrimSpace(s)))
}
