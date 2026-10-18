package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/media"

	"gorm.io/gorm"
)

// mediaStore holds uploaded files. It is set at startup by InitMediaStore.
var mediaStore media.Store

// InitMediaStore sets the store used for uploads and media URLs.
func InitMediaStore(s media.Store) { mediaStore = s }

// MediaStore returns the configured store (nil when uploads are disabled).
func MediaStore() media.Store { return mediaStore }

// UploadPolicy limits what can be uploaded.
type UploadPolicy struct {
	MaxSize      int64    // bytes; 0 means no limit
	AllowedTypes []string // MIME patterns; empty means media.DefaultAllowedTypes
}

// MediaError is an upload refused for a reason the client can fix.
type MediaError struct {
	Status int // HTTP status to answer with
	Msg    string
}

func (e *MediaError) Error() string { return e.Msg }

// ErrMediaNotFound is returned for unknown media IDs.
var ErrMediaNotFound = errors.New("media not found")

// ErrNoMediaStore is returned when no store is configured.
var ErrNoMediaStore = errors.New("media uploads are not configured")

// UploadFile describes one file to upload.
type UploadFile struct {
	Name    string        // client file name
	Size    int64         // bytes
	Content io.ReadSeeker // positioned at the start
	Alt     string
	Caption string
	User    string
}

// SaveMedia checks a file against the policy, stores its content and
// records it. The type is detected from the content, not from the client.
func SaveMedia(ctx context.Context, in UploadFile, policy UploadPolicy) (*models.Media, error) {
	if mediaStore == nil {
		return nil, ErrNoMediaStore
	}
	if policy.MaxSize > 0 && in.Size > policy.MaxSize {
		return nil, &MediaError{Status: 413, Msg: fmt.Sprintf("'%s' is %s; the limit is %s", in.Name, humanSize(in.Size), humanSize(policy.MaxSize))}
	}
	if in.Size == 0 {
		return nil, &MediaError{Status: 400, Msg: fmt.Sprintf("'%s' is empty", in.Name)}
	}

	head := make([]byte, 512)
	n, err := io.ReadFull(in.Content, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("cannot read upload: %w", err)
	}
	mimeType := media.DetectType(head[:n], in.Name)
	allowed := policy.AllowedTypes
	if len(allowed) == 0 {
		allowed = media.DefaultAllowedTypes
	}
	if !media.Allowed(mimeType, allowed) {
		return nil, &MediaError{Status: 415, Msg: fmt.Sprintf("'%s' is %s, which is not an allowed type", in.Name, mimeType)}
	}

	// Hash the content, and read image dimensions from the header.
	if _, err := in.Content.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, in.Content); err != nil {
		return nil, fmt.Errorf("cannot read upload: %w", err)
	}
	m := &models.Media{
		Name:       cleanFileName(in.Name),
		MimeType:   mimeType,
		Size:       in.Size,
		Hash:       hex.EncodeToString(hasher.Sum(nil)),
		Alt:        in.Alt,
		Caption:    in.Caption,
		UploadedBy: in.User,
		Driver:     mediaStore.Driver(),
	}
	if strings.HasPrefix(mimeType, "image/") {
		if _, err := in.Content.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		if w, h, ok := media.ImageSize(in.Content); ok {
			m.Width, m.Height = &w, &h
		}
	}
	if _, err := in.Content.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	m.Key = newMediaKey(m.Name)
	meta := media.ObjectMeta{ContentType: mimeType, Attachment: !media.InlineSafe(mimeType), FileName: m.Name}
	if err := mediaStore.Put(ctx, m.Key, in.Content, in.Size, meta); err != nil {
		return nil, fmt.Errorf("cannot store '%s': %w", in.Name, err)
	}
	if err := database.DB.Create(m).Error; err != nil {
		// Do not leave an orphan object behind.
		if derr := mediaStore.Delete(context.WithoutCancel(ctx), m.Key); derr != nil {
			logger.Log.WithError(derr).WithField("key", m.Key).Warn("Failed to remove stored file after a database error")
		}
		return nil, fmt.Errorf("cannot record '%s': %w", in.Name, err)
	}
	return m, nil
}

var unsafeNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// cleanFileName keeps the base name only, without control characters.
func cleanFileName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." || name == ".." || name == "/" {
		return "file"
	}
	if len(name) > 200 {
		// Keep the end (with the extension), cut on a character boundary.
		start := len(name) - 200
		for start < len(name) && !utf8.RuneStart(name[start]) {
			start++
		}
		name = name[start:]
	}
	return name
}

// newMediaKey builds "YYYY/MM/<random>-<slug>.<ext>": unguessable, unique
// and readable.
func newMediaKey(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if !regexp.MustCompile(`^\.[a-z0-9]{1,10}$`).MatchString(ext) {
		ext = ""
	}
	slug := strings.Trim(unsafeNameChars.ReplaceAllString(strings.TrimSuffix(name, filepath.Ext(name)), "-"), "-.")
	slug = strings.ToLower(slug)
	if len(slug) > 60 {
		slug = slug[:60]
	}
	if slug == "" {
		slug = "file"
	}
	rnd := make([]byte, 8)
	_, _ = rand.Read(rnd)
	return fmt.Sprintf("%s/%s-%s%s", time.Now().UTC().Format("2006/01"), hex.EncodeToString(rnd), slug, ext)
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}

// MediaURL returns the download URL of a media file.
func MediaURL(ctx context.Context, m models.Media) string {
	if mediaStore == nil {
		return ""
	}
	u, err := mediaStore.URL(ctx, m.Key)
	if err != nil {
		logger.Log.WithError(err).WithField("media_id", m.ID).Warn("Failed to build media URL")
		return ""
	}
	return u
}

// GetMedia returns a media file by ID.
func GetMedia(id uint) (*models.Media, error) {
	var m models.Media
	if err := database.DB.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMediaNotFound
		}
		return nil, err
	}
	return &m, nil
}

// MediaQuery filters the media library.
type MediaQuery struct {
	Page, PageSize int
	MimePrefix     string // e.g. "image/" or "application/pdf"
	Search         string // substring of the file name
}

// ListMedia lists media files, newest first.
func ListMedia(q MediaQuery) ([]models.Media, int, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 25
	}
	base := database.DB.Model(&models.Media{})
	if q.MimePrefix != "" {
		base = base.Where("mime_type LIKE ?"+likeEscape(), escapeLike(strings.ToLower(q.MimePrefix))+"%")
	}
	if q.Search != "" {
		base = base.Where("LOWER(name) LIKE ?"+likeEscape(), "%"+escapeLike(strings.ToLower(q.Search))+"%")
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []models.Media
	if err := base.Session(&gorm.Session{}).Order("id DESC").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, int(total), nil
}

// likeEscape is the ESCAPE clause matching escapeLike for the dialect.
func likeEscape() string {
	if database.DB.Dialector.Name() == "mysql" {
		return ` ESCAPE '\\'` // MySQL treats backslash as an escape inside string literals
	}
	return ` ESCAPE '\'`
}

// MediaInfo holds the editable fields of a media file.
type MediaInfo struct {
	Name    *string
	Alt     *string
	Caption *string
}

// UpdateMedia changes a media file's name, alternative text or caption.
func UpdateMedia(id uint, info MediaInfo) (*models.Media, error) {
	m, err := GetMedia(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if info.Name != nil {
		updates["name"] = cleanFileName(*info.Name)
	}
	if info.Alt != nil {
		updates["alt"] = *info.Alt
	}
	if info.Caption != nil {
		updates["caption"] = *info.Caption
	}
	if len(updates) > 0 {
		if err := database.DB.Model(m).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return GetMedia(id)
}

// DeleteMedia removes a media file and its content. Items that referenced
// it then show null, like a deleted related item. The record goes first, so
// a failure to delete the content leaves an orphan object (logged), never a
// record pointing at nothing.
func DeleteMedia(ctx context.Context, id uint) error {
	m, err := GetMedia(id)
	if err != nil {
		return err
	}
	if err := database.DB.Unscoped().Delete(m).Error; err != nil {
		return err
	}
	if mediaStore != nil {
		if err := mediaStore.Delete(context.WithoutCancel(ctx), m.Key); err != nil {
			logger.Log.WithError(err).WithField("key", m.Key).Warn("Media record deleted but its stored file could not be removed")
		}
	}
	return nil
}

// ---------------------------------------------------------------- reads

// componentCache keeps component definitions for media population, so
// reading items does not query components every time. Local writes clear
// it; other replicas see changes within componentCacheTTL.
var componentCache = struct {
	sync.Mutex
	at     time.Time
	byName map[string]*models.Component
}{byName: map[string]*models.Component{}}

const componentCacheTTL = 30 * time.Second

// InvalidateComponentCache drops cached component definitions.
func InvalidateComponentCache() {
	componentCache.Lock()
	componentCache.byName = map[string]*models.Component{}
	componentCache.Unlock()
}

func cachedComponent(name string) *models.Component {
	componentCache.Lock()
	defer componentCache.Unlock()
	if time.Since(componentCache.at) > componentCacheTTL {
		componentCache.byName = map[string]*models.Component{}
		componentCache.at = time.Now()
	}
	if c, ok := componentCache.byName[name]; ok {
		return c
	}
	c, err := GetComponentByName(name)
	if err != nil {
		c = nil
	}
	componentCache.byName[name] = c
	return c
}

// componentHasMedia reports whether a component (or one nested in it) has
// media attributes.
func componentHasMedia(name string, depth int) bool {
	c := cachedComponent(name)
	if c == nil || depth > 5 {
		return false
	}
	for _, a := range c.Attributes {
		if a.Type == "media" || (a.Type == "component" && componentHasMedia(a.ComponentRef, depth+1)) {
			return true
		}
	}
	return false
}

// hasMediaFields reports whether population has anything to do.
func hasMediaFields(attrs []models.Attribute) bool {
	for _, a := range attrs {
		if a.Type == "media" || (a.Type == "component" && componentHasMedia(a.ComponentRef, 1)) {
			return true
		}
	}
	return false
}

// mediaResolver turns stored media values into API objects. It loads all
// the files a value set references with one query.
type mediaResolver struct {
	ctx   context.Context
	media map[uint]map[string]any // nil value: deleted file
}

func newMediaResolver(ctx context.Context) *mediaResolver {
	return &mediaResolver{ctx: ctx, media: map[uint]map[string]any{}}
}

// mediaID returns the file ID a stored value references (0 for URLs).
func mediaID(v any) uint {
	if s, ok := v.(string); ok && !isDigits(s) {
		return 0
	}
	return models.ToUint(v)
}

func listOf(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []uint:
		out := make([]any, len(t))
		for i, id := range t {
			out[i] = id
		}
		return out
	}
	return nil
}

// load fetches the files not loaded yet.
func (r *mediaResolver) load(ids []uint) {
	var missing []uint
	for _, id := range ids {
		if _, ok := r.media[id]; !ok && id != 0 {
			missing = append(missing, id)
			r.media[id] = nil
		}
	}
	if len(missing) == 0 {
		return
	}
	var files []models.Media
	if err := database.DB.Where("id IN ?", missing).Find(&files).Error; err != nil {
		logger.Log.WithError(err).Warn("Failed to load media")
		return
	}
	for _, m := range files {
		r.media[m.ID] = models.MediaResponse(m, MediaURL(r.ctx, m))
	}
}

// value resolves one stored value: an ID becomes the media object (nil if
// it was deleted), a URL string is returned as is.
func (r *mediaResolver) value(v any) any {
	if s, ok := v.(string); ok && !isDigits(s) {
		return s
	}
	id := models.ToUint(v)
	if id == 0 {
		return nil
	}
	r.load([]uint{id})
	if obj := r.media[id]; obj != nil {
		return obj
	}
	return nil
}

func (r *mediaResolver) field(v any, multiple bool) any {
	if !multiple {
		return r.value(v)
	}
	list := listOf(v)
	ids := make([]uint, 0, len(list))
	for _, el := range list {
		ids = append(ids, mediaID(el))
	}
	r.load(ids)
	out := []any{}
	for _, el := range list {
		if resolved := r.value(el); resolved != nil {
			out = append(out, resolved)
		}
	}
	return out
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// walk applies fn to every media value of attrs in data, descending into
// components, and returns the data with fn's results.
func (r *mediaResolver) walk(attrs []models.Attribute, data map[string]any, depth int, fn func(v any, multiple bool) any) map[string]any {
	if depth > 5 || data == nil {
		return data
	}
	out := make(map[string]any, len(data))
	for k, v := range data {
		out[k] = v
	}
	for _, attr := range attrs {
		v, ok := data[attr.Name]
		if !ok || v == nil {
			continue
		}
		switch attr.Type {
		case "media":
			out[attr.Name] = fn(v, attr.Multiple)
		case "component":
			if !componentHasMedia(attr.ComponentRef, 1) {
				continue
			}
			cmp := cachedComponent(attr.ComponentRef)
			sub := make([]models.Attribute, 0, len(cmp.Attributes))
			for _, a := range cmp.Attributes {
				sub = append(sub, a.ToAttribute())
			}
			if attr.Repeatable {
				list, _ := v.([]any)
				nl := make([]any, 0, len(list))
				for _, el := range list {
					if obj, ok := el.(map[string]any); ok {
						nl = append(nl, r.walk(sub, obj, depth+1, fn))
					} else {
						nl = append(nl, el)
					}
				}
				out[attr.Name] = nl
			} else if obj, ok := v.(map[string]any); ok {
				out[attr.Name] = r.walk(sub, obj, depth+1, fn)
			}
		}
	}
	return out
}

// populate collects every referenced file, loads them in one query, then
// replaces the values.
func (r *mediaResolver) populate(attrs []models.Attribute, data map[string]any) map[string]any {
	var ids []uint
	r.walk(attrs, data, 0, func(v any, multiple bool) any {
		if multiple {
			for _, el := range listOf(v) {
				ids = append(ids, mediaID(el))
			}
		} else {
			ids = append(ids, mediaID(v))
		}
		return v
	})
	r.load(ids)
	return r.walk(attrs, data, 0, r.field)
}

// PopulateMedia replaces media IDs in an item's data (including inside
// components) with media objects.
func PopulateMedia(ctx context.Context, attrs []models.Attribute, data models.JSONMap) models.JSONMap {
	if !hasMediaFields(attrs) {
		return data
	}
	return models.JSONMap(newMediaResolver(ctx).populate(attrs, data))
}

// ResolveMediaValue converts one stored media value (an ID, a list of IDs,
// or URL strings) to API objects; GraphQL uses it.
func ResolveMediaValue(ctx context.Context, v any, multiple bool) any {
	return newMediaResolver(ctx).field(v, multiple)
}
