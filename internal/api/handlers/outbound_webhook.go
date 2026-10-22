package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/internal/webhooks"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// webhookInput is the body of webhook create and update requests. Fields
// left out of an update keep their value.
type webhookInput struct {
	Name         *string            `json:"name"`
	URL          *string            `json:"url"`
	Events       *[]string          `json:"events"`
	Collections  *[]string          `json:"collections"`
	Format       *string            `json:"format"`
	Headers      *map[string]string `json:"headers"`
	Enabled      *bool              `json:"enabled"`
	Signed       *bool              `json:"signed"`        // false: no signature (default true)
	RotateSecret bool               `json:"rotate_secret"` // update only
}

// apply validates in and copies it onto w. It reports whether a new secret
// was generated.
func (in webhookInput) apply(w *models.Webhook, creating bool) (bool, error) {
	if in.Name != nil {
		w.Name = strings.TrimSpace(*in.Name)
	}
	if w.Name == "" {
		return false, errors.New("name is required")
	}
	if in.URL != nil {
		w.URL = strings.TrimSpace(*in.URL)
	}
	if err := webhooks.ValidateURL(w.URL); err != nil {
		return false, err
	}
	if in.Events != nil {
		for _, e := range *in.Events {
			if e != "*" && !slices.Contains(models.WebhookEvents, e) {
				return false, fmt.Errorf("unknown event '%s' (expected one of %v or \"*\")", e, models.WebhookEvents)
			}
		}
		w.Events = *in.Events
	}
	if in.Collections != nil {
		w.Collections = *in.Collections
	}
	if in.Format != nil {
		w.Format = *in.Format
	}
	if w.Format == "" {
		w.Format = models.WebhookFormatJSON
	}
	if w.Format != models.WebhookFormatJSON && w.Format != models.WebhookFormatSlack {
		return false, fmt.Errorf("format must be 'json' or 'slack'")
	}
	if in.Headers != nil {
		for k := range *in.Headers {
			if err := webhooks.ValidateHeaderName(k); err != nil {
				return false, err
			}
		}
		w.Headers = *in.Headers
	}
	if in.Enabled != nil {
		w.Enabled = *in.Enabled
	} else if creating {
		w.Enabled = true
	}
	newSecret := false
	switch {
	case in.Signed != nil && !*in.Signed:
		w.Secret = ""
	case creating, in.RotateSecret, in.Signed != nil && *in.Signed && w.Secret == "":
		w.Secret = webhooks.NewSecret()
		newSecret = true
	}
	return newSecret, nil
}

func webhookJSON(w models.Webhook, showSecret bool) map[string]any {
	out := models.WebhookResponse(w)
	if showSecret {
		out["secret"] = w.Secret
	}
	return out
}

// CreateWebhook registers a webhook. The signing secret is returned once.
func CreateWebhook(c *gin.Context) {
	var in webhookInput
	if err := c.ShouldBindJSON(&in); err != nil {
		validationError(c, "Invalid JSON body")
		return
	}
	var w models.Webhook
	shown, err := in.apply(&w, true)
	if err != nil {
		validationError(c, err.Error())
		return
	}
	if err := storage.CreateWebhook(&w); err != nil {
		c.Set("response", "Failed to save webhook")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", webhookJSON(w, shown))
	if shown {
		c.Set("meta", gin.H{"warning": "Store the secret now: it cannot be shown again."})
	}
	c.Set("status", http.StatusCreated)
}

// ListWebhooks lists webhooks.
func ListWebhooks(c *gin.Context) {
	list, err := storage.ListWebhooks()
	if err != nil {
		c.Set("response", "Failed to list webhooks")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, w := range list {
		out = append(out, demoURL(c, webhookJSON(w, false)))
	}
	c.Set("response", out)
	c.Set("meta", gin.H{"total": len(out)})
	c.Set("status", http.StatusOK)
}

func loadWebhook(c *gin.Context) (*models.Webhook, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		validationError(c, "Invalid webhook ID")
		return nil, false
	}
	w, err := storage.GetWebhook(uint(id))
	if err != nil {
		if errors.Is(err, storage.ErrWebhookNotFound) {
			c.Set("response", "Webhook not found")
			c.Set("status", http.StatusNotFound)
		} else {
			c.Set("response", "Failed to load webhook")
			c.Set("status", http.StatusInternalServerError)
		}
		return nil, false
	}
	return w, true
}

// GetWebhook returns one webhook.
func GetWebhook(c *gin.Context) {
	if w, ok := loadWebhook(c); ok {
		c.Set("response", demoURL(c, webhookJSON(*w, false)))
		c.Set("status", http.StatusOK)
	}
}

// UpdateWebhook changes a webhook; "rotate_secret": true issues a new secret.
func UpdateWebhook(c *gin.Context) {
	w, ok := loadWebhook(c)
	if !ok {
		return
	}
	var in webhookInput
	if err := c.ShouldBindJSON(&in); err != nil {
		validationError(c, "Invalid JSON body")
		return
	}
	shown, err := in.apply(w, false)
	if err != nil {
		validationError(c, err.Error())
		return
	}
	if err := storage.SaveWebhook(w); err != nil {
		c.Set("response", "Failed to save webhook")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", webhookJSON(*w, shown))
	c.Set("status", http.StatusOK)
}

// DeleteWebhook removes a webhook and its delivery log.
func DeleteWebhook(c *gin.Context) {
	w, ok := loadWebhook(c)
	if !ok {
		return
	}
	if err := storage.DeleteWebhook(w.ID); err != nil {
		c.Set("response", "Failed to delete webhook")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", nil)
	c.Set("status", http.StatusOK)
}

// ListWebhookDeliveries returns the newest delivery attempts (limit ≤ 100).
func ListWebhookDeliveries(c *gin.Context) {
	w, ok := loadWebhook(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	list, err := storage.ListWebhookDeliveries(w.ID, limit)
	if err != nil {
		c.Set("response", "Failed to list deliveries")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", list)
	c.Set("status", http.StatusOK)
}

// TestWebhook sends a "webhook:test" event right away and reports the
// outcome, so the URL, headers and signature can be checked from the admin.
func TestWebhook(c *gin.Context) {
	w, ok := loadWebhook(c)
	if !ok {
		return
	}
	d := webhooks.Delivery{
		WebhookID:  w.ID,
		DeliveryID: fmt.Sprintf("test-%d-%d", w.ID, time.Now().UnixNano()),
		Event:      webhooks.EventTest,
		Collection: "",
		Item:       map[string]any{"message": "This is a test delivery from GoHead."},
		OccurredAt: time.Now().UTC(),
	}
	res := webhooks.Send(c.Request.Context(), *w, d, 1)
	out := gin.H{"delivered": res.Err == nil, "statusCode": res.StatusCode, "durationMs": res.Duration.Milliseconds()}
	if res.Err != nil {
		out["error"] = res.Err.Error()
	}
	c.Set("response", out)
	c.Set("status", http.StatusOK)
}
