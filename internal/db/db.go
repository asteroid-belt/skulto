// Package db provides a GORM-based database layer for Skulto.
// It uses the pure-Go SQLite driver with FTS5 support.
package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/asteroid-belt/skulto/internal/models"
)

// DB wraps the GORM database connection with Skulto-specific operations.
type DB struct {
	*gorm.DB
	path string
}

// Config holds database configuration options.
type Config struct {
	Path        string
	Debug       bool
	MaxIdleConn int
	MaxOpenConn int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig(path string) Config {
	return Config{
		Path:        path,
		Debug:       false,
		MaxIdleConn: 1,
		MaxOpenConn: 1,
	}
}

// New creates a new database connection and runs migrations.
func New(cfg Config) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	// Configure GORM logger
	logLevel := logger.Silent
	if cfg.Debug {
		logLevel = logger.Info
	}

	// Build DSN with DELETE journal mode for simpler transaction handling
	// (WAL mode has visibility issues with the pure-Go SQLite driver)
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(DELETE)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", cfg.Path)

	// Open database with pure-Go SQLite driver (FTS5 enabled by default)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                 logger.Default.LogMode(logLevel),
		SkipDefaultTransaction: true, // Better performance for read operations
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConn)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConn)
	sqlDB.SetConnMaxLifetime(time.Hour)

	wrapped := &DB{DB: db, path: cfg.Path}

	// Run auto-migrations
	if err := wrapped.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	// Create FTS5 virtual table and triggers
	if err := wrapped.setupFTS(); err != nil {
		return nil, fmt.Errorf("setup FTS: %w", err)
	}

	// Seed default sync metadata
	if err := wrapped.seedSyncMeta(); err != nil {
		return nil, fmt.Errorf("seed sync meta: %w", err)
	}

	// Seed default onboarding state
	if err := wrapped.seedOnboarding(); err != nil {
		return nil, fmt.Errorf("seed onboarding state: %w", err)
	}

	// Ensure special tags exist
	if err := wrapped.EnsureMineTag(); err != nil {
		return nil, fmt.Errorf("ensure mine tag: %w", err)
	}

	return wrapped, nil
}

// migrate runs GORM auto-migrations for all models.
func (db *DB) migrate() error {
	return db.AutoMigrate(
		&models.Source{},
		&models.Tag{},
		&models.Skill{},
		&models.Installed{},
		&models.SyncMeta{},
		&models.UserState{},
		&models.SkillInstallation{},
		&models.AuxiliaryFile{},
		&models.SecurityScan{},
	)
}

// setupFTS creates the FTS5 virtual table and triggers for full-text search.
func (db *DB) setupFTS() error {
	// Create FTS5 virtual table if it doesn't exist
	ftsSQL := `
		CREATE VIRTUAL TABLE IF NOT EXISTS skills_fts USING fts5(
			title,
			description,
			content,
			summary,
			author,
			content='skills',
			content_rowid='rowid',
			tokenize='porter unicode61'
		);
	`
	if err := db.Exec(ftsSQL).Error; err != nil {
		return fmt.Errorf("create FTS table: %w", err)
	}

	// Create triggers to keep FTS in sync
	triggers := []string{
		// After INSERT
		`CREATE TRIGGER IF NOT EXISTS skills_ai AFTER INSERT ON skills BEGIN
			INSERT INTO skills_fts(rowid, title, description, content, summary, author)
			VALUES (NEW.rowid, NEW.title, NEW.description, NEW.content, NEW.summary, NEW.author);
		END;`,

		// After DELETE
		`CREATE TRIGGER IF NOT EXISTS skills_ad AFTER DELETE ON skills BEGIN
			INSERT INTO skills_fts(skills_fts, rowid, title, description, content, summary, author)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.description, OLD.content, OLD.summary, OLD.author);
		END;`,

		// After UPDATE
		`CREATE TRIGGER IF NOT EXISTS skills_au AFTER UPDATE ON skills BEGIN
			INSERT INTO skills_fts(skills_fts, rowid, title, description, content, summary, author)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.description, OLD.content, OLD.summary, OLD.author);
			INSERT INTO skills_fts(rowid, title, description, content, summary, author)
			VALUES (NEW.rowid, NEW.title, NEW.description, NEW.content, NEW.summary, NEW.author);
		END;`,
	}

	for _, trigger := range triggers {
		if err := db.Exec(trigger).Error; err != nil {
			return fmt.Errorf("create trigger: %w", err)
		}
	}

	return nil
}

// seedSyncMeta inserts default sync metadata if not present.
func (db *DB) seedSyncMeta() error {
	defaults := []models.SyncMeta{
		{Key: models.SyncMetaLastFullSync, Value: ""},
		{Key: models.SyncMetaLastDeltaSync, Value: ""},
		{Key: models.SyncMetaSchemaVersion, Value: "1"},
		{Key: models.SyncMetaTotalSkills, Value: "0"},
	}

	for _, meta := range defaults {
		// Only insert if not exists
		result := db.Where("key = ?", meta.Key).FirstOrCreate(&meta)
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

// seedOnboarding inserts default state if not present.
func (db *DB) seedOnboarding() error {
	defaultState := models.UserState{
		ID:               "default",
		OnboardingStatus: models.OnboardingNotStarted,
		AITools:          "", // Empty by default
	}

	// Only insert if not exists
	result := db.Where("id = ?", "default").FirstOrCreate(&defaultState)
	return result.Error
}

// Path returns the database file path.
func (db *DB) Path() string {
	return db.path
}

// Close closes the database connection.
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Transaction executes a function within a database transaction.
// The callback receives a *DB wrapper that uses the transaction.
// If the callback returns an error, the transaction is rolled back.
// If the callback returns nil, the transaction is committed.
func (d *DB) Transaction(fc func(tx *DB) error) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		wrappedTx := &DB{DB: tx, path: d.path}
		return fc(wrappedTx)
	})
}

// GetStats returns aggregate statistics about the database.
func (db *DB) GetStats() (*models.SkillStats, error) {
	var stats models.SkillStats

	if err := db.Model(&models.Skill{}).Count(&stats.TotalSkills).Error; err != nil {
		return nil, fmt.Errorf("count skills: %w", err)
	}

	if err := db.Model(&models.Tag{}).Count(&stats.TotalTags).Error; err != nil {
		return nil, fmt.Errorf("count tags: %w", err)
	}

	if err := db.Model(&models.Source{}).Count(&stats.TotalSources).Error; err != nil {
		return nil, fmt.Errorf("count sources: %w", err)
	}

	// Get database file size
	if info, err := os.Stat(db.path); err == nil {
		stats.CacheSizeBytes = info.Size()
	}

	stats.LastUpdated = time.Now()

	return &stats, nil
}
