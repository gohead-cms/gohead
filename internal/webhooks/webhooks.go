// Package webhooks delivers content events to external URLs. The event
// dispatcher fans each event out into one queued delivery per subscribed
// webhook; workers send them, signed, with retries and a delivery log.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/hibiken/asynq"
)

const (
	// TaskTypeDeliver is the queued task that sends one event to one webhook.
	TaskTypeDeliver = "webhook:deliver"
	// Queue is the queue deliveries go through (served by `gohead worker`).
	Queue = "webhooks"
	// EventTest is the event sent by the "test" endpoint.
	EventTest = "webhook:test"
)

// Config controls deliveries.
type Config struct {
	// AllowPrivateNetworks lets webhooks reach private and loopback
	// addresses (internal rebuild hooks, local development). Link-local
	// addresses (cloud metadata endpoints) are always refused.
	AllowPrivateNetworks bool
	Timeout              time.Duration
	MaxRetries           int
}

var cfg = Config{Timeout: 10 * time.Second, MaxRetries: 8}

// Configure sets the delivery settings (call at startup).
func Configure(c Config) {
	if c.Timeout <= 0 {
		c.Timeout = 10 * time.Second
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}
	cfg = c
}

// Delivery is the payload of a queued delivery.
type Delivery struct {
	WebhookID  uint           `json:"webhook_id"`
	DeliveryID string         `json:"delivery_id"`
	Event      string         `json:"event"`
	Collection string         `json:"collection"`
	ItemID     uint           `json:"item_id"`
	Item       map[string]any `json:"item"`
	OccurredAt time.Time      `json:"occurred_at"`
}

// Fanout enqueues one delivery per webhook subscribed to the event. The
// task IDs derive from the event task's ID, so a retried event does not
// deliver twice.
func Fanout(ctx context.Context, client *asynq.Client, ev events.CollectionEventPayload) error {
	hooks, err := storage.WebhooksFor(string(ev.EventType), ev.CollectionName)
	if err != nil || len(hooks) == 0 {
		return err
	}
	eventID, ok := asynq.GetTaskID(ctx)
	if !ok {
		eventID = newID()
	}
	item := map[string]any{}
	for k, v := range ev.ItemData {
		item[k] = v
	}
	if ev.ItemID != 0 {
		item["id"] = ev.ItemID
	}
	for _, w := range hooks {
		d := Delivery{
			WebhookID:  w.ID,
			DeliveryID: fmt.Sprintf("%s-%d", eventID, w.ID),
			Event:      string(ev.EventType),
			Collection: ev.CollectionName,
			ItemID:     ev.ItemID,
			Item:       item,
			OccurredAt: time.Now().UTC(),
		}
		body, err := json.Marshal(d)
		if err != nil {
			return err
		}
		_, err = client.EnqueueContext(ctx, asynq.NewTask(TaskTypeDeliver, body),
			asynq.Queue(Queue), asynq.TaskID("wh-"+d.DeliveryID),
			asynq.MaxRetry(cfg.MaxRetries), asynq.Timeout(cfg.Timeout+5*time.Second))
		if err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) {
			return fmt.Errorf("enqueue delivery for webhook %d: %w", w.ID, err)
		}
	}
	return nil
}

// HandleDelivery sends a queued delivery. Network errors, 5xx, 408 and 429
// are retried with backoff; other 4xx answers are not.
func HandleDelivery(ctx context.Context, t *asynq.Task) error {
	var d Delivery
	if err := json.Unmarshal(t.Payload(), &d); err != nil {
		return fmt.Errorf("bad delivery payload: %v: %w", err, asynq.SkipRetry)
	}
	w, err := storage.GetWebhook(d.WebhookID)
	if errors.Is(err, storage.ErrWebhookNotFound) {
		return nil // deleted since
	}
	if err != nil {
		return err
	}
	if !w.Enabled {
		return nil
	}
	attempt, _ := asynq.GetRetryCount(ctx)
	res := Send(ctx, *w, d, attempt+1)
	if res.Err == nil {
		return nil
	}
	if res.Permanent {
		return fmt.Errorf("%v: %w", res.Err, asynq.SkipRetry)
	}
	return res.Err
}

// Result describes one attempt.
type Result struct {
	StatusCode int
	Duration   time.Duration
	Err        error
	Permanent  bool // retrying will not help
}

// Send delivers d to w now, records the attempt and returns its outcome.
func Send(ctx context.Context, w models.Webhook, d Delivery, attempt int) Result {
	start := time.Now()
	res := send(ctx, w, d)
	res.Duration = time.Since(start)
	rec := &models.WebhookDelivery{
		WebhookID: w.ID, DeliveryID: d.DeliveryID, Event: d.Event, Collection: d.Collection,
		ItemID: d.ItemID, Attempt: attempt, StatusCode: res.StatusCode, DurationMs: res.Duration.Milliseconds(),
	}
	if res.Err != nil {
		rec.Error = truncate(res.Err.Error(), 500)
		logger.Log.WithError(res.Err).WithField("webhook", w.Name).WithField("attempt", attempt).Warn("Webhook delivery failed")
	}
	storage.RecordWebhookDelivery(rec)
	return res
}

func send(ctx context.Context, w models.Webhook, d Delivery) Result {
	body, err := Body(w, d)
	if err != nil {
		return Result{Err: err, Permanent: true}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return Result{Err: fmt.Errorf("invalid URL: %w", err), Permanent: true}
	}
	for k, v := range w.Headers {
		req.Header.Set(k, v)
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoHead-Webhooks/1")
	req.Header.Set("X-Gohead-Event", d.Event)
	req.Header.Set("X-Gohead-Delivery", d.DeliveryID)
	req.Header.Set("X-Gohead-Timestamp", ts)
	if w.Secret != "" {
		req.Header.Set("X-Gohead-Signature", Sign(w.Secret, ts, body))
	}

	resp, err := Client().Do(req)
	if err != nil {
		var blocked *BlockedAddressError
		return Result{Err: err, Permanent: errors.As(err, &blocked)}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10)) // reuse the connection
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return Result{StatusCode: resp.StatusCode}
	}
	retry := resp.StatusCode >= 500 || resp.StatusCode == http.StatusRequestTimeout || resp.StatusCode == http.StatusTooManyRequests
	return Result{StatusCode: resp.StatusCode, Err: fmt.Errorf("endpoint answered %d", resp.StatusCode), Permanent: !retry}
}

// Body renders the request body for the webhook's format.
func Body(w models.Webhook, d Delivery) ([]byte, error) {
	if w.Format == models.WebhookFormatSlack {
		return json.Marshal(map[string]string{"text": slackText(d)})
	}
	return json.Marshal(map[string]any{
		"id":         d.DeliveryID,
		"event":      d.Event,
		"collection": d.Collection,
		"occurredAt": d.OccurredAt.UTC().Format(time.RFC3339),
		"data":       d.Item,
	})
}

var slackVerbs = map[string]string{
	"item:created": "created", "item:updated": "updated", "item:deleted": "deleted",
	"item:published": "published", "item:unpublished": "unpublished", EventTest: "test",
}

// slackText summarizes an event in one line, naming the item by its
// title, name or slug when it has one.
func slackText(d Delivery) string {
	if d.Event == EventTest {
		return "GoHead: test message from webhook delivery " + d.DeliveryID
	}
	label := fmt.Sprintf("#%d", d.ItemID)
	for _, k := range []string{"title", "name", "slug", "label"} {
		if s, ok := d.Item[k].(string); ok && s != "" {
			label = fmt.Sprintf("%q (#%d)", truncate(s, 80), d.ItemID)
			break
		}
	}
	verb := slackVerbs[d.Event]
	if verb == "" {
		verb = d.Event
	}
	return fmt.Sprintf("GoHead: %s %s in %s", label, verb, d.Collection)
}

// Sign returns the X-Gohead-Signature value: HMAC-SHA256 of
// "<timestamp>.<body>" with the webhook's secret.
func Sign(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// NewSecret returns a random signing secret.
func NewSecret() string {
	return "whsec_" + newID() + newID()
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ---------------------------------------------------------------- network

// BlockedAddressError is returned when a webhook resolves to an address it
// may not reach.
type BlockedAddressError struct{ IP net.IP }

func (e *BlockedAddressError) Error() string {
	return fmt.Sprintf("address %s is not allowed for webhooks (see webhooks.allow_private_networks)", e.IP)
}

var privateBlocks = mustCIDRs(
	"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10", // RFC 1918, CGNAT
	"fc00::/7", // unique local IPv6
)

func mustCIDRs(cidrs ...string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		out = append(out, n)
	}
	return out
}

// AddressAllowed reports whether a webhook may connect to ip.
func AddressAllowed(ip net.IP, allowPrivate bool) bool {
	switch {
	case ip.IsUnspecified(), ip.IsMulticast(), ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast(), ip.IsInterfaceLocalMulticast():
		return false // includes 169.254.169.254 (cloud metadata)
	case ip.IsLoopback():
		return allowPrivate
	}
	for _, n := range privateBlocks {
		if n.Contains(ip) {
			return allowPrivate
		}
	}
	return true
}

// Client returns the HTTP client used for deliveries. Every connection,
// including after redirects, is checked once the name is resolved, so DNS
// tricks cannot reach forbidden addresses.
func Client() *http.Client {
	allowPrivate := cfg.AllowPrivateNetworks
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || !AddressAllowed(ip, allowPrivate) {
				return &BlockedAddressError{IP: ip}
			}
			return nil
		},
	}
	transport := &http.Transport{
		Proxy:                 nil, // a proxy would bypass the address check
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: cfg.Timeout,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
	}
	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
}

// reservedHeaders cannot be set per webhook.
var reservedHeaders = map[string]bool{
	"content-type": true, "content-length": true, "host": true, "user-agent": true,
	"transfer-encoding": true, "connection": true,
}

// ValidateHeaderName rejects headers GoHead sets itself.
func ValidateHeaderName(name string) error {
	l := strings.ToLower(strings.TrimSpace(name))
	if l == "" || reservedHeaders[l] || strings.HasPrefix(l, "x-gohead-") {
		return fmt.Errorf("header '%s' cannot be set", name)
	}
	for _, r := range l {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return fmt.Errorf("invalid header name '%s'", name)
		}
	}
	return nil
}

// ValidateURL checks a webhook URL's shape (addresses are checked at
// delivery time, after DNS resolution).
func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("url must be an absolute http(s) URL")
	}
	if u.User != nil {
		return fmt.Errorf("put credentials in headers, not in the URL")
	}
	return nil
}
