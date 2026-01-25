package db

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/asteroid-belt/skulto/internal/models"
)

// GetSyncMeta retrieves a sync metadata value.
func (db *DB) GetSyncMeta(key string) (string, error) {
	var meta models.SyncMeta
	err := db.First(&meta, "key = ?", key).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return meta.Value, nil
}

// SetSyncMeta sets a sync metadata value.
func (db *DB) SetSyncMeta(key, value string) error {
	meta := models.SyncMeta{Key: key, Value: value}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&meta).Error
}

// GetAllSyncMeta retrieves all sync metadata.
func (db *DB) GetAllSyncMeta() (map[string]string, error) {
	var metas []models.SyncMeta
	if err := db.Find(&metas).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, meta := range metas {
		result[meta.Key] = meta.Value
	}
	return result, nil
}

// DeleteSyncMeta deletes a sync metadata entry.
func (db *DB) DeleteSyncMeta(key string) error {
	return db.Delete(&models.SyncMeta{}, "key = ?", key).Error
}
