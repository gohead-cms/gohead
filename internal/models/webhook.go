package models

import (
	"slices"
	"time"

	"gorm.io/gorm"
)

// Webhook formats.
const (
	WebhookFormatJSON  = "json"  // the full event as JSON
	WebhookFormatSlack = "slack" // {"text": "..."}, for Slack (and compatible) incoming webhooks
)

// WebhookEvents are the events a webhook can subscribe to ("*" means all).
var WebhookEvents = []string{"item:created", "item:updated", "item:deleted", "item:published", "item:unpublished", "singleton:updated", "singleton:deleted"}

// Webhook sends content events to an external URL: an integration such as
// Slack or Zapier, or a static-site rebuild hook.
type Webhook struct {
	gorm.Model
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Events      []string          `json:"events" gorm:"type:json;serializer:json"`      // empty or "*": all
	Collections []string          `json:"collections" gorm:"type:json;serializer:json"` // empty: all
	Format      string            `json:"format"`                                       // json | slack
	Headers     map[string]string `json:"-" gorm:"type:json;serializer:json"`           // sent with each delivery
	Secret      string            `json:"-"`                                            // HMAC signing key
	Enabled     bool              `json:"enabled" gorm:"not null"`                      // no DB default: GORM would override false on create

	LastStatus     int        `json:"lastStatus"`
	LastError      string     `json:"lastError"`
	LastDeliveryAt *time.Time `json:"lastDeliveryAt"`
}

// Matches reports whether the webhook wants event on collection. Only the
// WebhookEvents are delivered: "all events" never includes internal ones
// such as schema changes.
func (w Webhook) Matches(event, collection string) bool {
	if !w.Enabled || !slices.Contains(WebhookEvents, event) {
		return false
	}
	return matchList(w.Events, event) && matchList(w.Collections, collection)
}

func matchList(list []string, v string) bool {
	if len(list) == 0 {
		return true
	}
	for _, x := range list {
		if x == "*" || x == v {
			return true
		}
	}
	return false
}

// WebhookResponse is the JSON shape of a webhook. Header values and the
// secret are never returned (they often hold credentials).
func WebhookResponse(w Webhook) map[string]any {
	headers := make([]string, 0, len(w.Headers))
	for k := range w.Headers {
		headers = append(headers, k)
	}
	var last any
	if w.LastDeliveryAt != nil {
		last = APITime(*w.LastDeliveryAt)
	}
	events, collections := w.Events, w.Collections
	if events == nil {
		events = []string{}
	}
	if collections == nil {
		collections = []string{}
	}
	return map[string]any{
		"id":             w.ID,
		"name":           w.Name,
		"url":            w.URL,
		"events":         events,
		"collections":    collections,
		"format":         w.Format,
		"headers":        headers, // names only
		"signed":         w.Secret != "",
		"enabled":        w.Enabled,
		"lastStatus":     w.LastStatus,
		"lastError":      w.LastError,
		"lastDeliveryAt": last,
		"createdAt":      APITime(w.CreatedAt),
		"updatedAt":      APITime(w.UpdatedAt),
	}
}

// WebhookDelivery records one delivery attempt.
type WebhookDelivery struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	WebhookID  uint      `json:"webhookId" gorm:"index"`
	DeliveryID string    `json:"deliveryId" gorm:"index"` // same for every attempt of one event
	Event      string    `json:"event"`
	Collection string    `json:"collection"`
	ItemID     uint      `json:"itemId"`
	Attempt    int       `json:"attempt"`
	StatusCode int       `json:"statusCode"` // 0 when no response was received
	Error      string    `json:"error"`
	DurationMs int64     `json:"durationMs"`
	CreatedAt  time.Time `json:"createdAt"`
}
