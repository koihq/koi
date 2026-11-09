// Package posthogprovider sends Koi's server events to PostHog.
package posthogprovider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/posthog/posthog-go"
)

const DefaultHost = "https://eu.i.posthog.com"

// EmailLookup returns a user's email. The client uses it to skip excluded accounts.
type EmailLookup func(ctx context.Context, userID uuid.UUID) (string, error)

type Client struct {
	client   posthog.Client
	excluded map[string]struct{}
	lookup   EmailLookup

	mu      sync.Mutex
	skipped map[uuid.UUID]bool
}

// New starts a client that queues events and sends them in batches. Accounts whose email is in
// excludedEmails, such as the team's own, send nothing.
func New(key, host string, excludedEmails []string, lookup EmailLookup) (*Client, error) {
	client, err := posthog.NewWithConfig(key, posthog.Config{
		Endpoint:        host,
		Interval:        5 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		Logger:          slogLogger{},
	})
	if err != nil {
		return nil, fmt.Errorf("posthog client: %w", err)
	}

	excluded := make(map[string]struct{}, len(excludedEmails))
	for _, email := range excludedEmails {
		if email = strings.ToLower(strings.TrimSpace(email)); email != "" {
			excluded[email] = struct{}{}
		}
	}

	return &Client{
		client:   client,
		excluded: excluded,
		lookup:   lookup,
		skipped:  map[uuid.UUID]bool{},
	}, nil
}

func (c *Client) Track(userID, organizationID uuid.UUID, event string, props map[string]any) {
	if userID == uuid.Nil || c.isExcluded(userID) {
		return
	}

	properties := posthog.NewProperties()
	for key, value := range props {
		properties.Set(key, value)
	}

	if organizationID != uuid.Nil {
		properties.Set("organization_id", organizationID.String())
	}

	err := c.client.Enqueue(posthog.Capture{
		DistinctId: userID.String(),
		Event:      event,
		Properties: properties,
	})
	if err != nil {
		slog.Warn("analytics event dropped", "event", event, "err", err)
	}
}

// isExcluded looks each user up once per process. A failed lookup sends the event: the PostHog
// test-account filter still hides it.
func (c *Client) isExcluded(userID uuid.UUID) bool {
	if len(c.excluded) == 0 || c.lookup == nil {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if skip, ok := c.skipped[userID]; ok {
		return skip
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	email, err := c.lookup(ctx, userID)
	if err != nil {
		return false
	}

	_, skip := c.excluded[strings.ToLower(email)]
	c.skipped[userID] = skip

	return skip
}

// Close sends what is still queued.
func (c *Client) Close() error {
	return c.client.Close()
}

type slogLogger struct{}

func (slogLogger) Debugf(format string, args ...any) {
	slog.Debug("posthog: " + fmt.Sprintf(format, args...))
}

func (slogLogger) Logf(format string, args ...any) {
	slog.Debug("posthog: " + fmt.Sprintf(format, args...))
}

func (slogLogger) Warnf(format string, args ...any) {
	slog.Warn("posthog: " + fmt.Sprintf(format, args...))
}

func (slogLogger) Errorf(format string, args ...any) {
	slog.Warn("posthog: " + fmt.Sprintf(format, args...))
}
