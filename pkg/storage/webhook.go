package storage

import (
	"errors"
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// ErrWebhookNotFound is returned for unknown webhook IDs.
var ErrWebhookNotFound = errors.New("webhook not found")

// deliveriesKept is how many delivery records are kept per webhook.
const deliveriesKept = 100

// CreateWebhook stores a webhook.
func CreateWebhook(w *models.Webhook) error {
	return database.DB.Create(w).Error
}

// SaveWebhook updates every field of a webhook.
func SaveWebhook(w *models.Webhook) error {
	return database.DB.Save(w).Error
}

// GetWebhook returns a webhook by ID.
func GetWebhook(id uint) (*models.Webhook, error) {
	var w models.Webhook
	if err := database.DB.First(&w, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWebhookNotFound
		}
		return nil, err
	}
	return &w, nil
}

// ListWebhooks returns every webhook, oldest first.
func ListWebhooks() ([]models.Webhook, error) {
	var out []models.Webhook
	err := database.DB.Order("id ASC").Find(&out).Error
	return out, err
}

// WebhooksFor returns the enabled webhooks subscribed to event on collection.
func WebhooksFor(event, collection string) ([]models.Webhook, error) {
	var all []models.Webhook
	if err := database.DB.Where("enabled = ?", true).Find(&all).Error; err != nil {
		return nil, err
	}
	var out []models.Webhook
	for _, w := range all {
		if w.Matches(event, collection) {
			out = append(out, w)
		}
	}
	return out, nil
}

// DeleteWebhook removes a webhook and its delivery log.
func DeleteWebhook(id uint) error {
	if _, err := GetWebhook(id); err != nil {
		return err
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("webhook_id = ?", id).Delete(&models.WebhookDelivery{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&models.Webhook{}, id).Error
	})
}

// RecordWebhookDelivery logs an attempt, updates the webhook's last status
// and trims old records.
func RecordWebhookDelivery(d *models.WebhookDelivery) {
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now()
	}
	if err := database.DB.Create(d).Error; err != nil {
		logger.Log.WithError(err).Warn("Failed to record webhook delivery")
		return
	}
	if err := database.DB.Model(&models.Webhook{}).Where("id = ?", d.WebhookID).Updates(map[string]any{
		"last_status": d.StatusCode, "last_error": d.Error, "last_delivery_at": d.CreatedAt,
	}).Error; err != nil {
		logger.Log.WithError(err).Warn("Failed to update webhook status")
	}
	// Keep the newest records only.
	var cutoff models.WebhookDelivery
	err := database.DB.Where("webhook_id = ?", d.WebhookID).Order("id DESC").Offset(deliveriesKept).First(&cutoff).Error
	if err == nil {
		database.DB.Where("webhook_id = ? AND id <= ?", d.WebhookID, cutoff.ID).Delete(&models.WebhookDelivery{})
	}
}

// ListWebhookDeliveries returns the newest delivery records of a webhook.
func ListWebhookDeliveries(webhookID uint, limit int) ([]models.WebhookDelivery, error) {
	if limit < 1 || limit > deliveriesKept {
		limit = deliveriesKept
	}
	var out []models.WebhookDelivery
	err := database.DB.Where("webhook_id = ?", webhookID).Order("id DESC").Limit(limit).Find(&out).Error
	return out, err
}
