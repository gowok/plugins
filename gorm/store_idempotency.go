package gorm

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/ngamux/middleware/idempotency"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultIdempotencyTable = "gowok_idempotency"
	defaultIdempotencyTTL   = 24 * time.Hour
	idempotencyTimeout      = 5 * time.Second
)

// StoreIdempotency implements idempotency.Store on top of a GORM connection.
// The zero value is ready to use and stores entries through the default
// connection.
type StoreIdempotency struct {
	// Name selects the GORM connection. Empty means "default".
	Name string
	// Table overrides the table used to store idempotency entries.
	Table string
	// TTL is how long completed responses are kept. Defaults to 24 hours.
	TTL time.Duration
}

var _ idempotency.Store = StoreIdempotency{}

type idempotencyRecord struct {
	Key        string `gorm:"column:idempotency_key;primaryKey;size:255"`
	InProgress bool   `gorm:"column:in_progress;not null;default:false"`
	Status     int    `gorm:"column:status;not null;default:0"`
	Header     string `gorm:"column:header;type:text"`
	Body       []byte `gorm:"column:body"`
	ExpiresAt  int64  `gorm:"column:expires_at;not null;default:0;index"`
}

type idempotencyMigrationKey struct {
	db    *gorm.DB
	table string
}

var (
	idempotencyMigrateMu sync.Mutex
	idempotencyMigrated  sync.Map
)

func (s StoreIdempotency) Begin(key string) (*idempotency.StoredResponse, idempotency.State) {
	db, ok := s.database()
	if !ok {
		return nil, idempotency.StateMissing
	}

	if err := s.migrate(db); err != nil {
		slog.Warn("failed to prepare idempotency table", "plugin", plugin, "error", err)
		return nil, idempotency.StateMissing
	}

	ctx, cancel := context.WithTimeout(context.Background(), idempotencyTimeout)
	defer cancel()

	now := time.Now().Unix()
	record := idempotencyRecord{Key: key, InProgress: true}

	result := db.WithContext(ctx).
		Table(s.tableName()).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&record)
	if result.Error != nil {
		slog.Warn("failed to start idempotency entry", "plugin", plugin, "error", result.Error)
		return nil, idempotency.StateMissing
	}
	if result.RowsAffected == 1 {
		return nil, idempotency.StateMissing
	}

	// The key already exists. Decide whether it is still running, replayable,
	// or expired and safe to claim.
	for attempt := 0; attempt < 3; attempt++ {
		var existing idempotencyRecord
		err := db.WithContext(ctx).
			Table(s.tableName()).
			Where("idempotency_key = ?", key).
			Take(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result = db.WithContext(ctx).
				Table(s.tableName()).
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&record)
			if result.Error != nil {
				slog.Warn("failed to start idempotency entry", "plugin", plugin, "error", result.Error)
				return nil, idempotency.StateMissing
			}
			if result.RowsAffected == 1 {
				return nil, idempotency.StateMissing
			}
			continue
		}
		if err != nil {
			slog.Warn("failed to read idempotency entry", "plugin", plugin, "error", err)
			return nil, idempotency.StateMissing
		}

		if existing.InProgress {
			return nil, idempotency.StateInProgress
		}
		if existing.ExpiresAt > now {
			return existing.response(), idempotency.StateFound
		}

		// Expired: claim it atomically. The in_progress = false condition makes
		// sure only one concurrent request wins the claim.
		result = db.WithContext(ctx).
			Table(s.tableName()).
			Where(
				"idempotency_key = ? AND in_progress = ? AND (expires_at = 0 OR expires_at <= ?)",
				key, false, now,
			).
			Updates(map[string]any{
				"in_progress": true,
				"status":      0,
				"header":      "",
				"body":        nil,
				"expires_at":  0,
			})
		if result.Error != nil {
			slog.Warn("failed to claim expired idempotency entry", "plugin", plugin, "error", result.Error)
			return nil, idempotency.StateMissing
		}
		if result.RowsAffected == 1 {
			return nil, idempotency.StateMissing
		}
	}

	// A concurrent request kept changing the row; reject the request rather
	// than risk processing it twice.
	return nil, idempotency.StateInProgress
}

func (s StoreIdempotency) Complete(key string, response *idempotency.StoredResponse) {
	if response == nil {
		s.Abort(key)
		return
	}

	db, ok := s.database()
	if !ok {
		return
	}

	if err := s.migrate(db); err != nil {
		slog.Warn("failed to prepare idempotency table", "plugin", plugin, "error", err)
		return
	}

	header, err := json.Marshal(response.Header)
	if err != nil {
		slog.Warn("failed to encode idempotency header", "plugin", plugin, "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), idempotencyTimeout)
	defer cancel()

	result := db.WithContext(ctx).
		Table(s.tableName()).
		Where("idempotency_key = ? AND in_progress = ?", key, true).
		Updates(map[string]any{
			"in_progress": false,
			"status":      response.Status,
			"header":      string(header),
			"body":        response.Body,
			"expires_at":  time.Now().Add(s.ttl()).Unix(),
		})
	if result.Error != nil {
		slog.Warn("failed to store idempotency response", "plugin", plugin, "error", result.Error)
	}
}

func (s StoreIdempotency) Abort(key string) {
	db, ok := s.database()
	if !ok {
		return
	}

	if err := s.migrate(db); err != nil {
		slog.Warn("failed to prepare idempotency table", "plugin", plugin, "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), idempotencyTimeout)
	defer cancel()

	result := db.WithContext(ctx).
		Table(s.tableName()).
		Where("idempotency_key = ? AND in_progress = ?", key, true).
		Delete(&idempotencyRecord{})
	if result.Error != nil {
		slog.Warn("failed to abort idempotency entry", "plugin", plugin, "error", result.Error)
	}
}

func (s StoreIdempotency) ttl() time.Duration {
	if s.TTL > 0 {
		return s.TTL
	}
	return defaultIdempotencyTTL
}

func (s StoreIdempotency) tableName() string {
	if s.Table != "" {
		return s.Table
	}
	return defaultIdempotencyTable
}

func (s StoreIdempotency) database() (*gorm.DB, bool) {
	name := s.Name
	if name == "" {
		name = "default"
	}

	db, ok := DB(name).Get()
	if !ok || db == nil {
		slog.Warn("no connection available", "name", name, "plugin", plugin)
		return nil, false
	}

	return db, true
}

func (s StoreIdempotency) migrate(db *gorm.DB) error {
	table := s.tableName()
	key := idempotencyMigrationKey{db: db, table: table}
	if _, ok := idempotencyMigrated.Load(key); ok {
		return nil
	}

	idempotencyMigrateMu.Lock()
	defer idempotencyMigrateMu.Unlock()

	if _, ok := idempotencyMigrated.Load(key); ok {
		return nil
	}

	if err := db.Table(table).AutoMigrate(&idempotencyRecord{}); err != nil {
		return err
	}

	idempotencyMigrated.Store(key, struct{}{})
	return nil
}

func (r idempotencyRecord) response() *idempotency.StoredResponse {
	response := &idempotency.StoredResponse{
		Status: r.Status,
		Body:   r.Body,
	}

	if r.Header != "" {
		if err := json.Unmarshal([]byte(r.Header), &response.Header); err != nil {
			slog.Warn("failed to decode idempotency header", "plugin", plugin, "error", err)
		}
	}

	return response
}
