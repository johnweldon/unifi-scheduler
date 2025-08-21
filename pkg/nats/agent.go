package nats

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
)

func NewAgent(s *unifi.Session, base string, opts ...ClientOpt) *Agent {
	addnl := []ClientOpt{
		OptBuckets(DetailBucket(base), ByMACBucket(base), ByNameBucket(base)),
		OptStreams(EventStream(base)),
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

	// Perform health checks before starting
	if err := a.healthCheck(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	go a.serve(ctx)

	return nil
}

func (a *Agent) healthCheck(ctx context.Context) error {
	log.Printf("performing health checks...")

	// Test UniFi connection
	if err := a.client.Initialize(); err != nil {
		return fmt.Errorf("unifi initialization failed: %w", err)
	}

	if _, err := a.client.Login(); err != nil {
		return fmt.Errorf("unifi login failed: %w", err)
	}

	// Test NATS connection
	if err := a.publisher.ensureConnection(); err != nil {
		return fmt.Errorf("nats connection failed: %w", err)
	}

	log.Printf("health checks passed")
	return nil
}

func (a *Agent) serve(ctx context.Context) {
	var err error

	eventInterval := time.After(1 * time.Second)
	lookupInterval := time.After(7 * time.Second)
	clientInterval := time.After(11 * time.Second)
	userInterval := time.After(19 * time.Second)
	deviceInterval := time.After(47 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return

		case <-eventInterval:
			eventInterval = time.After(37 * time.Second)
			if err = a.publishEventsWithContext(ctx); err != nil {
				log.Printf("error: publishing events %v", err)
			}

		case <-clientInterval:
			clientInterval = time.After(53 * time.Second)
			if err = a.refreshClientsWithContext(ctx); err != nil {
				log.Printf("error: refreshing clients %v", err)
			}

		case <-userInterval:
			userInterval = time.After(337 * time.Second)
			if err = a.refreshUsersWithContext(ctx); err != nil {
				log.Printf("error: refreshing users %v", err)
			}

		case <-deviceInterval:
			deviceInterval = time.After(607 * time.Second)
			if err = a.refreshDevicesWithContext(ctx); err != nil {
				log.Printf("error: refreshing devices %v", err)
			}

		case <-lookupInterval:
			lookupInterval = time.After(997 * time.Second)
			if err = a.refreshLookupsWithContext(ctx); err != nil {
				log.Printf("error: refreshing lookups %v", err)
			}
		}
	}
}

func (a *Agent) publishEvents() error {
	return a.publishEventsWithContext(context.Background())
}

func (a *Agent) publishEventsWithContext(ctx context.Context) error {
	return a.withRetry(ctx, "publishEvents", func(ctx context.Context) error {
		events, err := a.client.GetRecentEvents()
		if err != nil {
			return fmt.Errorf("get events: %w", err)
		}

		for _, evt := range events {
			if err = a.publishStreamWithContext(ctx, EventStream(a.base), string(evt.Key), evt); err != nil {
				return fmt.Errorf("publish events: %w", err)
			}
		}

		const maxEvents = 500
		if len(events) > maxEvents {
			events = events[len(events)-500:]
		}

		if err = a.storeWithContext(ctx, DetailBucket(a.base), EventsKey, events); err != nil {
			return fmt.Errorf("persisting recent events: %w", err)
		}

		return nil
	})
}

func (a *Agent) refreshClients() error {
	return a.refreshClientsWithContext(context.Background())
}

func (a *Agent) refreshClientsWithContext(ctx context.Context) error {
	return a.withRetry(ctx, "refreshClients", func(ctx context.Context) error {
		clients, err := a.client.GetClients()
		if err != nil {
			return fmt.Errorf("get clients: %w", err)
		}

		if err = a.publish("clients", clients); err != nil {
			return err
		}

		if err = a.storeWithContext(ctx, DetailBucket(a.base), ActiveKey, clients); err != nil {
			return fmt.Errorf("persisting live clients: %w", err)
		}

		return nil
	})
}

func (a *Agent) refreshUsers() error {
	return a.refreshUsersWithContext(context.Background())
}

func (a *Agent) refreshUsersWithContext(ctx context.Context) error {
	return a.withRetry(ctx, "refreshUsers", func(ctx context.Context) error {
		users, err := a.client.GetAllClients()
		if err != nil {
			return fmt.Errorf("get users: %w", err)
		}

		if err = a.publish("users", users); err != nil {
			return err
		}

		for _, user := range users {
			mac := user.MAC.String()
			if err = a.storeWithContext(ctx, DetailBucket(a.base), mac, user); err != nil {
				return fmt.Errorf("persisting user %q: %w", mac, err)
			}
		}

		return nil
	})
}

func (a *Agent) refreshDevices() error {
	return a.refreshDevicesWithContext(context.Background())
}

func (a *Agent) refreshDevicesWithContext(ctx context.Context) error {
	return a.withRetry(ctx, "refreshDevices", func(ctx context.Context) error {
		devices, err := a.client.GetDevices()
		if err != nil {
			return fmt.Errorf("get devices: %w", err)
		}

		if err = a.publish(DevicesSubject, devices); err != nil {
			return err
		}

		for _, device := range devices {
			mac := device.MAC.String()
			if err = a.storeWithContext(ctx, DetailBucket(a.base), mac, device); err != nil {
				return fmt.Errorf("persisting device %q: %w", mac, err)
			}
		}

		if err = a.storeWithContext(ctx, DetailBucket(a.base), DevicesKey, devices); err != nil {
			return fmt.Errorf("persisting devices: %w", err)
		}

		return nil
	})
}

func (a *Agent) refreshLookups() error {
	return a.refreshLookupsWithContext(context.Background())
}

func (a *Agent) refreshLookupsWithContext(ctx context.Context) error {
	return a.withRetry(ctx, "refreshLookups", func(ctx context.Context) error {
		macs, err := a.client.GetMACs()
		if err != nil {
			return fmt.Errorf("get MACs: %w", err)
		}

		for k, v := range macs {
			mac := k.String()
			if err = a.storeWithContext(ctx, ByMACBucket(a.base), mac, v); err != nil {
				return fmt.Errorf("persisting MAC names %q: %w", mac, err)
			}
		}

		names, err := a.client.GetNames()
		if err != nil {
			return fmt.Errorf("get names: %w", err)
		}

		for k, v := range names {
			if err = a.storeWithContext(ctx, ByNameBucket(a.base), k, v); err != nil {
				return fmt.Errorf("persisting name MACs %q: %w", k, err)
			}
		}

		return nil
	})
}

func (a *Agent) publish(subject string, msg any) error {
	if err := a.publisher.Publish(subSubject(a.base, subject), msg); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

func (a *Agent) publishStream(stream, subject string, msg any) error {
	return a.publishStreamWithContext(context.Background(), stream, subject, msg)
}

func (a *Agent) publishStreamWithContext(ctx context.Context, stream, subject string, msg any) error {
	if err := a.publisher.PublishStreamWithContext(ctx, stream, subject, msg); err != nil {
		return fmt.Errorf("publish stream %q: %w", stream, err)
	}

	return nil
}

func (a *Agent) store(bucket, key string, val any) error {
	return a.storeWithContext(context.Background(), bucket, key, val)
}

func (a *Agent) storeWithContext(ctx context.Context, bucket, key string, val any) error {
	norm := NormalizeKey(key)
	if norm == "" {
		return nil
	}

	if err := a.publisher.StoreWithContext(ctx, bucket, norm, val); err != nil {
		return fmt.Errorf("storing key %q: %w", norm, err)
	}

	return nil
}

const (
	ActiveKey  = "active"
	DevicesKey = "devices"
	EventsKey  = "events"

	DevicesSubject = "devices"
)

func DetailBucket(base string) string        { return base + "-details" }
func ByMACBucket(base string) string         { return base + "-bymac" }
func ByNameBucket(base string) string        { return base + "-byname" }
func EventStream(base string) string         { return base + "-events" }
func subSubject(base, subject string) string { return base + "." + subject }

func (a *Agent) withRetry(ctx context.Context, operation string, fn func(context.Context) error) error {
	const maxRetries = 3
	baseDelay := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		if attempt == maxRetries {
			return fmt.Errorf("%s: final attempt failed after %d retries: %w", operation, maxRetries, err)
		}

		// Exponential backoff with jitter
		delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		log.Printf("warning: %s failed (attempt %d/%d), retrying in %v: %v",
			operation, attempt+1, maxRetries+1, delay, err)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Continue to next attempt
		}
	}

	return fmt.Errorf("unreachable")
}

func NormalizeKey(s string) string {
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
