package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logger.InitLogger("error")
	logger.Log.SetOutput(&bytes.Buffer{})
}

func setup(t *testing.T) {
	t.Helper()
	_, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, migrations.MigrateDatabase(db))
	prev := cfg
	Configure(Config{AllowPrivateNetworks: true, Timeout: 2 * time.Second, MaxRetries: 3})
	t.Cleanup(func() { cfg = prev })
}

type received struct {
	headers http.Header
	body    []byte
}

func receiver(t *testing.T, status int) (*httptest.Server, func() []received) {
	t.Helper()
	var mu sync.Mutex
	var got []received
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, received{r.Header.Clone(), b})
		mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []received { mu.Lock(); defer mu.Unlock(); return append([]received(nil), got...) }
}

func TestSendSignsAndRecords(t *testing.T) {
	setup(t)
	srv, got := receiver(t, http.StatusOK)
	w := models.Webhook{Name: "site", URL: srv.URL + "/hook", Format: models.WebhookFormatJSON,
		Secret: "whsec_test", Enabled: true, Headers: map[string]string{"Authorization": "Bearer abc"}}
	require.NoError(t, storage.CreateWebhook(&w))

	d := Delivery{WebhookID: w.ID, DeliveryID: "evt-1", Event: "item:published", Collection: "articles",
		ItemID: 7, Item: map[string]any{"id": 7, "title": "Hello"}, OccurredAt: time.Now()}
	res := Send(context.Background(), w, d, 1)
	require.NoError(t, res.Err)
	assert.Equal(t, 200, res.StatusCode)

	reqs := got()
	require.Len(t, reqs, 1)
	r := reqs[0]
	assert.Equal(t, "item:published", r.headers.Get("X-Gohead-Event"))
	assert.Equal(t, "evt-1", r.headers.Get("X-Gohead-Delivery"))
	assert.Equal(t, "Bearer abc", r.headers.Get("Authorization"), "custom headers are sent")
	ts := r.headers.Get("X-Gohead-Timestamp")
	_, err := strconv.ParseInt(ts, 10, 64)
	require.NoError(t, err)
	assert.Equal(t, Sign("whsec_test", ts, r.body), r.headers.Get("X-Gohead-Signature"), "receivers can verify the signature")

	var body map[string]any
	require.NoError(t, json.Unmarshal(r.body, &body))
	assert.Equal(t, "item:published", body["event"])
	assert.Equal(t, "articles", body["collection"])
	assert.Equal(t, "Hello", body["data"].(map[string]any)["title"])

	log, err := storage.ListWebhookDeliveries(w.ID, 10)
	require.NoError(t, err)
	require.Len(t, log, 1)
	assert.Equal(t, 200, log[0].StatusCode)
	stored, _ := storage.GetWebhook(w.ID)
	assert.Equal(t, 200, stored.LastStatus)
	assert.NotNil(t, stored.LastDeliveryAt)
}

func TestRetryClassification(t *testing.T) {
	setup(t)
	for status, permanent := range map[int]bool{500: false, 503: false, 429: false, 408: false, 404: true, 400: true, 410: true} {
		srv, _ := receiver(t, status)
		w := models.Webhook{Name: "x", URL: srv.URL, Enabled: true}
		require.NoError(t, storage.CreateWebhook(&w))
		res := Send(context.Background(), w, Delivery{WebhookID: w.ID, Event: "item:created"}, 1)
		require.Error(t, res.Err, "%d", status)
		assert.Equal(t, permanent, res.Permanent, "%d", status)
		assert.Equal(t, status, res.StatusCode)
	}
}

func TestSlackFormat(t *testing.T) {
	setup(t)
	srv, got := receiver(t, http.StatusOK)
	w := models.Webhook{Name: "slack", URL: srv.URL, Format: models.WebhookFormatSlack, Enabled: true}
	require.NoError(t, storage.CreateWebhook(&w))
	Send(context.Background(), w, Delivery{WebhookID: w.ID, Event: "item:published", Collection: "articles", ItemID: 3, Item: map[string]any{"title": "Launch day"}}, 1)
	var body map[string]string
	require.NoError(t, json.Unmarshal(got()[0].body, &body))
	assert.Equal(t, `GoHead: "Launch day" (#3) published in articles`, body["text"])
	assert.Empty(t, got()[0].headers.Get("X-Gohead-Signature"), "no secret, no signature")
}

func TestAddressGuard(t *testing.T) {
	cases := map[string][2]bool{ // ip -> {allowed by default, allowed with private networks}
		"8.8.8.8":         {true, true},
		"127.0.0.1":       {false, true},
		"10.1.2.3":        {false, true},
		"192.168.1.10":    {false, true},
		"100.64.0.1":      {false, true},
		"169.254.169.254": {false, false}, // cloud metadata: never
		"::1":             {false, true},
		"fe80::1":         {false, false},
		"fd00::1":         {false, true},
		"0.0.0.0":         {false, false},
	}
	for ip, want := range cases {
		assert.Equal(t, want[0], AddressAllowed(net.ParseIP(ip), false), ip)
		assert.Equal(t, want[1], AddressAllowed(net.ParseIP(ip), true), ip+" (private allowed)")
	}

	setup(t)
	Configure(Config{AllowPrivateNetworks: false, Timeout: time.Second})
	srv, got := receiver(t, http.StatusOK)
	w := models.Webhook{Name: "internal", URL: srv.URL, Enabled: true}
	require.NoError(t, storage.CreateWebhook(&w))
	res := Send(context.Background(), w, Delivery{WebhookID: w.ID, Event: "item:created"}, 1)
	require.Error(t, res.Err)
	assert.True(t, res.Permanent, "a blocked address is not retried")
	assert.Contains(t, res.Err.Error(), "not allowed for webhooks")
	assert.Empty(t, got(), "nothing reached the loopback receiver")
}

func TestValidation(t *testing.T) {
	assert.NoError(t, ValidateURL("https://hooks.slack.com/services/T/B/X"))
	for _, bad := range []string{"ftp://x", "/relative", "https://user:pw@example.com", "not a url", ""} {
		assert.Error(t, ValidateURL(bad), bad)
	}
	assert.NoError(t, ValidateHeaderName("Authorization"))
	assert.NoError(t, ValidateHeaderName("X-Api-Key"))
	for _, bad := range []string{"Content-Type", "Host", "X-Gohead-Signature", "Bad Header", ""} {
		assert.Error(t, ValidateHeaderName(bad), bad)
	}
}

// Fan-out enqueues one delivery per matching webhook. (Deduplication of a
// retried event relies on the event task's ID, which only asynq sets.)
func TestFanout(t *testing.T) {
	setup(t)
	mr := miniredis.RunT(t)
	opt := asynq.RedisClientOpt{Addr: mr.Addr()}
	client := asynq.NewClient(opt)
	defer client.Close()
	inspector := asynq.NewInspector(opt)
	defer inspector.Close()

	all := models.Webhook{Name: "all", URL: "https://example.com/a", Enabled: true}
	onlyPublished := models.Webhook{Name: "rebuild", URL: "https://example.com/b", Enabled: true, Events: []string{"item:published"}, Collections: []string{"articles"}}
	disabled := models.Webhook{Name: "off", URL: "https://example.com/c"}
	for _, w := range []*models.Webhook{&all, &onlyPublished, &disabled} {
		require.NoError(t, storage.CreateWebhook(w))
	}
	require.NoError(t, storage.SaveWebhook(func() *models.Webhook { disabled.Enabled = false; return &disabled }()))

	ev := events.CollectionEventPayload{EventType: "item:published", CollectionName: "articles", ItemID: 9, ItemData: map[string]any{"title": "x"}}
	ctx := context.Background()
	require.NoError(t, Fanout(ctx, client, ev))
	tasks, err := inspector.ListPendingTasks(Queue)
	require.NoError(t, err)
	assert.Len(t, tasks, 2, "all + rebuild, not the disabled one")

	other := events.CollectionEventPayload{EventType: "item:updated", CollectionName: "pages", ItemID: 1}
	require.NoError(t, Fanout(ctx, client, other))
	tasks, _ = inspector.ListPendingTasks(Queue)
	assert.Len(t, tasks, 3, "only 'all' matches item:updated on pages")

	schema := events.CollectionEventPayload{EventType: "collection:created", CollectionName: "pages"}
	require.NoError(t, Fanout(ctx, client, schema))
	tasks, _ = inspector.ListPendingTasks(Queue)
	assert.Len(t, tasks, 3, "internal events are never delivered, even to 'all events' webhooks")

	var d Delivery
	require.NoError(t, json.Unmarshal(tasks[0].Payload, &d))
	assert.Equal(t, float64(9), d.Item["id"], "the item ID is part of the data")
}
