// pkg/storage/content_type.go
package storage

import (
	"sync"

	"gitlab.com/sudo.bngz/gohead/internal/models"
)

var (
	contentTypes = make(map[string]models.ContentType)
	ctMutex      sync.RWMutex
)

func SaveContentType(ct models.ContentType) {
	ctMutex.Lock()
	defer ctMutex.Unlock()
	contentTypes[ct.Name] = ct
}

func GetContentType(name string) (models.ContentType, bool) {
	ctMutex.RLock()
	defer ctMutex.RUnlock()
	ct, exists := contentTypes[name]
	return ct, exists
}
