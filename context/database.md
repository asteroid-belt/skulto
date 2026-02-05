# Database Deep Dive

> This document details the database schema and operations in Skulto.

## Overview

Skulto uses SQLite with GORM ORM and FTS5 for full-text search. The database is stored at `~/.skulto/skulto.db`.

## Database Configuration

```go
type DBConfig struct {
    Path            string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
    SlowThreshold   time.Duration
    LogLevel        string
}

func DefaultConfig(path string) *DBConfig {
    return &DBConfig{
        Path:            path,
        MaxOpenConns:    1,     // SQLite single-writer
        MaxIdleConns:    1,
        ConnMaxLifetime: 0,     // No timeout
        SlowThreshold:   200 * time.Millisecond,
        LogLevel:        "warn",
    }
}
```

## Schema

### Skills Table

```sql
CREATE TABLE skills (
    id            TEXT PRIMARY KEY,
    slug          TEXT,
    title         TEXT,
    description   TEXT,
    content       TEXT,
    summary       TEXT,
    source_id     TEXT,
    file_path     TEXT,
    category      TEXT,
    difficulty    TEXT DEFAULT 'intermediate',
    stars         INTEGER DEFAULT 0,
    forks         INTEGER DEFAULT 0,
    downloads     INTEGER DEFAULT 0,
    embedding_id  TEXT,
    version       TEXT,
    license       TEXT,
    author        TEXT,
    is_local      INTEGER DEFAULT 0,
    is_installed  INTEGER DEFAULT 0,
    security_status TEXT DEFAULT 'PENDING',
    threat_level    TEXT DEFAULT 'NONE',
    threat_summary  TEXT,
    scanned_at    DATETIME,
    released_at   DATETIME,
    content_hash  TEXT,
    created_at    DATETIME,
    updated_at    DATETIME,
    deleted_at    DATETIME,
    indexed_at    DATETIME,
    last_sync_at  DATETIME,
    viewed_at     DATETIME,

    FOREIGN KEY (source_id) REFERENCES sources(id)
);

CREATE UNIQUE INDEX idx_slug_source ON skills(slug, source_id);
CREATE INDEX idx_skills_title ON skills(title);
CREATE INDEX idx_skills_category ON skills(category);
CREATE INDEX idx_skills_difficulty ON skills(difficulty);
CREATE INDEX idx_skills_author ON skills(author);
CREATE INDEX idx_skills_is_installed ON skills(is_installed);
CREATE INDEX idx_skills_security_status ON skills(security_status);
```

### FTS5 Virtual Table

```sql
CREATE VIRTUAL TABLE skills_fts USING fts5(
    title,
    description,
    content,
    summary,
    tags,
    content='skills',
    content_rowid='rowid'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER skills_ai AFTER INSERT ON skills BEGIN
    INSERT INTO skills_fts(rowid, title, description, content, summary, tags)
    VALUES (NEW.rowid, NEW.title, NEW.description, NEW.content, NEW.summary, '');
END;

CREATE TRIGGER skills_ad AFTER DELETE ON skills BEGIN
    INSERT INTO skills_fts(skills_fts, rowid, title, description, content, summary, tags)
    VALUES ('delete', OLD.rowid, OLD.title, OLD.description, OLD.content, OLD.summary, '');
END;

CREATE TRIGGER skills_au AFTER UPDATE ON skills BEGIN
    INSERT INTO skills_fts(skills_fts, rowid, title, description, content, summary, tags)
    VALUES ('delete', OLD.rowid, OLD.title, OLD.description, OLD.content, OLD.summary, '');
    INSERT INTO skills_fts(rowid, title, description, content, summary, tags)
    VALUES (NEW.rowid, NEW.title, NEW.description, NEW.content, NEW.summary, '');
END;
```

### Tags Table

```sql
CREATE TABLE tags (
    id       TEXT PRIMARY KEY,
    name     TEXT,
    slug     TEXT,
    category TEXT,
    color    TEXT,
    count    INTEGER DEFAULT 0
);
```

### Skill-Tag Association

```sql
CREATE TABLE skill_tags (
    skill_id TEXT,
    tag_id   TEXT,
    PRIMARY KEY (skill_id, tag_id),
    FOREIGN KEY (skill_id) REFERENCES skills(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);
```

### Sources Table

```sql
CREATE TABLE sources (
    id           TEXT PRIMARY KEY,
    owner        TEXT,
    repo         TEXT,
    branch       TEXT DEFAULT 'main',
    skill_path   TEXT,
    url          TEXT,
    description  TEXT,
    stars        INTEGER DEFAULT 0,
    forks        INTEGER DEFAULT 0,
    license      TEXT,
    last_sync_at DATETIME,
    created_at   DATETIME,
    updated_at   DATETIME
);
```

### Skill Installations Table

```sql
CREATE TABLE skill_installations (
    id         TEXT PRIMARY KEY,
    skill_id   TEXT NOT NULL,
    platform   TEXT NOT NULL,
    scope      TEXT NOT NULL,
    path       TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (skill_id) REFERENCES skills(id)
);

CREATE INDEX idx_installations_skill ON skill_installations(skill_id);
CREATE INDEX idx_installations_platform ON skill_installations(platform);
```

### User State Table

```sql
CREATE TABLE user_state (
    id                    INTEGER PRIMARY KEY,
    tracking_id           TEXT,
    ai_tools              TEXT,
    onboarding_completed  INTEGER DEFAULT 0,
    created_at            DATETIME,
    updated_at            DATETIME
);
```

### Agent Preferences Table

```sql
CREATE TABLE agent_preferences (
    platform   TEXT PRIMARY KEY,
    enabled    INTEGER DEFAULT 0,
    scope      TEXT DEFAULT 'global',
    created_at DATETIME,
    updated_at DATETIME
);
```

### Sync Metadata Table

```sql
CREATE TABLE sync_meta (
    source_id  TEXT PRIMARY KEY,
    commit_sha TEXT,
    synced_at  DATETIME
);
```

### Auxiliary Files Table

```sql
CREATE TABLE auxiliary_files (
    id         TEXT PRIMARY KEY,
    skill_id   TEXT NOT NULL,
    path       TEXT NOT NULL,
    content    TEXT,
    created_at DATETIME,
    updated_at DATETIME,
    FOREIGN KEY (skill_id) REFERENCES skills(id)
);
```

### Discovered Skills Table

```sql
CREATE TABLE discovered_skills (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    path         TEXT NOT NULL,
    source       TEXT NOT NULL,
    description  TEXT,
    dismissed    INTEGER DEFAULT 0,
    created_at   DATETIME,
    updated_at   DATETIME
);
```

## GORM Models

### Skill Model

```go
type Skill struct {
    ID   string `gorm:"primaryKey;size:64"`
    Slug string `gorm:"uniqueIndex:idx_slug_source;size:100"`

    // Content
    Title       string `gorm:"size:255;index"`
    Description string `gorm:"size:1000"`
    Content     string `gorm:"type:text"`
    Summary     string `gorm:"size:500"`

    // Source information
    SourceID *string `gorm:"size:255;index;uniqueIndex:idx_slug_source"`
    Source   *Source `gorm:"foreignKey:SourceID"`
    FilePath string  `gorm:"size:500"`

    // Categorization
    Tags       []Tag  `gorm:"many2many:skill_tags"`
    Category   string `gorm:"size:50;index"`
    Difficulty string `gorm:"size:20;index;default:intermediate"`

    // Metrics
    Stars     int `gorm:"default:0"`
    Forks     int `gorm:"default:0"`
    Downloads int `gorm:"default:0"`

    // Embedding
    EmbeddingID string `gorm:"size:64"`

    // Metadata
    Version string `gorm:"size:50"`
    License string `gorm:"size:100"`
    Author  string `gorm:"size:255;index"`

    // Local state
    IsLocal     bool `gorm:"default:false"`
    IsInstalled bool `gorm:"default:false;index"`

    // Security
    SecurityStatus SecurityStatus `gorm:"size:20;default:PENDING;index"`
    ThreatLevel    ThreatLevel    `gorm:"size:20;default:NONE"`
    ThreatSummary  string         `gorm:"size:1000"`
    ScannedAt      *time.Time
    ContentHash    string `gorm:"size:64"`

    // Auxiliary files
    AuxiliaryFiles []AuxiliaryFile `gorm:"foreignKey:SkillID"`

    // Timestamps
    CreatedAt  time.Time
    UpdatedAt  time.Time
    DeletedAt  gorm.DeletedAt `gorm:"index"`
    IndexedAt  time.Time
    LastSyncAt *time.Time
    ViewedAt   *time.Time
}
```

## Key Database Operations

### FTS5 Search with BM25 Ranking

```go
func (db *DB) Search(query string, limit int) ([]SearchResult, error) {
    ftsQuery := prepareFTSQuery(query)

    var results []SearchResult
    err := db.Raw(`
        SELECT s.*, bm25(skills_fts, 10.0, 5.0, 1.0, 2.0, 3.0) as rank
        FROM skills s
        JOIN skills_fts fts ON s.rowid = fts.rowid
        WHERE skills_fts MATCH ?
          AND s.deleted_at IS NULL
        ORDER BY rank
        LIMIT ?
    `, ftsQuery, limit).Scan(&results).Error

    return results, err
}
```

BM25 weights: `title(10), description(5), content(1), summary(2), tags(3)`

### FTS Query Preparation

```go
func prepareFTSQuery(query string) string {
    terms := strings.Fields(query)
    var escaped []string

    for _, term := range terms {
        // Remove FTS5 special characters
        term = strings.ReplaceAll(term, "\"", "")
        term = strings.ReplaceAll(term, "'", "")
        term = strings.ReplaceAll(term, "(", "")
        term = strings.ReplaceAll(term, ")", "")
        term = strings.ReplaceAll(term, "*", "")
        term = strings.ReplaceAll(term, ":", "")
        term = strings.ReplaceAll(term, "-", " ")

        if term != "" {
            escaped = append(escaped, term+"*")  // Prefix matching
        }
    }

    return strings.Join(escaped, " ")
}
```

### Upsert with Selective Updates

```go
func (db *DB) UpsertSkill(skill *models.Skill) error {
    return db.Clauses(clause.OnConflict{
        Columns: []clause.Column{{Name: "id"}},
        DoUpdates: clause.AssignmentColumns([]string{
            // Update metadata fields
            "slug", "title", "description", "content", "summary",
            "source_id", "file_path", "category", "difficulty",
            "stars", "forks", "downloads", "embedding_id",
            "version", "license", "author",
            "indexed_at", "last_sync_at", "updated_at",
            // NOT updated: is_local, is_installed, viewed_at
            // NOT updated: security_status, threat_level, threat_summary, scanned_at
        }),
    }).Create(skill).Error
}
```

### Efficient Joins vs Preloads

```go
// Use Joins for one-to-one relationships (Source)
// Use Preload for many-to-many relationships (Tags)
func (db *DB) GetSkill(id string) (*models.Skill, error) {
    var skill models.Skill
    err := db.Joins("Source").
        Preload("Tags").
        Preload("AuxiliaryFiles").
        First(&skill, "skills.id = ?", id).Error
    return &skill, err
}
```

### Transaction with Tag Count Maintenance

```go
func (db *DB) UpsertSkillWithTags(skill *models.Skill, tags []models.Tag) error {
    return db.Transaction(func(tx *DB) error {
        // Get existing tags for decrement
        var existingSkill models.Skill
        skillExists := tx.Preload("Tags").First(&existingSkill, "id = ?", skill.ID).Error == nil

        oldTagIDs := make(map[string]bool)
        if skillExists {
            for _, tag := range existingSkill.Tags {
                oldTagIDs[tag.ID] = true
            }
        }

        // Upsert skill
        if err := tx.Clauses(...).Create(skill).Error; err != nil {
            return err
        }

        // Upsert tags and update counts
        newTagIDs := make(map[string]bool)
        for i := range tags {
            tag := &tags[i]
            newTagIDs[tag.ID] = true

            // Upsert tag
            tx.Clauses(...).Create(tag)

            // Increment if new
            if !oldTagIDs[tag.ID] {
                tx.Model(&models.Tag{}).Where("id = ?", tag.ID).
                    Update("count", gorm.Expr("count + 1"))
            }
        }

        // Decrement removed tags
        for tagID := range oldTagIDs {
            if !newTagIDs[tagID] {
                tx.Model(&models.Tag{}).Where("id = ?", tagID).
                    Update("count", gorm.Expr("CASE WHEN count > 0 THEN count - 1 ELSE 0 END"))
            }
        }

        // Replace associations
        return tx.Model(skill).Association("Tags").Replace(tags)
    })
}
```

## Migrations

Migrations are handled by GORM's AutoMigrate with custom extensions:

```go
func (db *DB) AutoMigrate() error {
    // Standard GORM migrations
    if err := db.DB.AutoMigrate(
        &models.Skill{},
        &models.Tag{},
        &models.Source{},
        &models.UserState{},
        &models.AgentPreference{},
        &models.SkillInstallation{},
        &models.SyncMeta{},
        &models.AuxiliaryFile{},
        &models.DiscoveredSkill{},
    ); err != nil {
        return err
    }

    // Create FTS5 table and triggers
    return db.createFTSTable()
}

func (db *DB) createFTSTable() error {
    // Check if FTS table exists
    var count int64
    db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='skills_fts'").Scan(&count)
    if count > 0 {
        return nil
    }

    // Create FTS5 virtual table
    sql := `CREATE VIRTUAL TABLE skills_fts USING fts5(
        title, description, content, summary, tags,
        content='skills', content_rowid='rowid'
    )`
    if err := db.Exec(sql).Error; err != nil {
        return err
    }

    // Create sync triggers
    // ... trigger SQL
    return nil
}
```

## Database Statistics

```go
type SkillStats struct {
    TotalSkills    int64
    TotalTags      int64
    TotalSources   int64
    LastUpdated    time.Time
    CacheSizeBytes int64
    EmbeddingCount int64
}

func (db *DB) GetStats() (*models.SkillStats, error) {
    var stats models.SkillStats

    db.Model(&models.Skill{}).Count(&stats.TotalSkills)
    db.Model(&models.Tag{}).Count(&stats.TotalTags)
    db.Model(&models.Source{}).Count(&stats.TotalSources)

    var lastUpdated time.Time
    db.Model(&models.Skill{}).Select("MAX(updated_at)").Scan(&lastUpdated)
    stats.LastUpdated = lastUpdated

    // Get embedding count
    db.Model(&models.Skill{}).
        Where("embedding_id IS NOT NULL AND embedding_id != ''").
        Count(&stats.EmbeddingCount)

    return &stats, nil
}
```

## Common Query Patterns

### Get Recent Skills

```go
func (db *DB) GetRecentSkills(limit int) ([]models.Skill, error) {
    var skills []models.Skill
    err := db.Preload("Tags").
        Where("viewed_at IS NOT NULL").
        Order("viewed_at DESC").
        Limit(limit).
        Find(&skills).Error
    return skills, err
}
```

### Get Skills by Tag

```go
func (db *DB) GetSkillsByTag(tagSlug string, limit, offset int) ([]models.Skill, error) {
    var skills []models.Skill
    err := db.Preload("Tags").
        Joins("JOIN skill_tags st ON skills.id = st.skill_id").
        Where("st.tag_id = ?", tagSlug).
        Order("skills.stars DESC, skills.updated_at DESC").
        Limit(limit).
        Offset(offset).
        Find(&skills).Error
    return skills, err
}
```

### Get Pending Security Scans

```go
func (db *DB) GetPendingSkills() ([]models.Skill, error) {
    var skills []models.Skill
    err := db.Where("security_status = ?", models.SecurityStatusPending).
        Find(&skills).Error
    return skills, err
}
```

### Record Installation

```go
func (db *DB) RecordInstallation(skillID string, loc installer.InstallLocation) error {
    inst := &models.SkillInstallation{
        ID:       fmt.Sprintf("%s-%s-%s", skillID, loc.Platform, loc.Scope),
        SkillID:  skillID,
        Platform: string(loc.Platform),
        Scope:    string(loc.Scope),
        Path:     loc.Path,
    }
    return db.Clauses(clause.OnConflict{UpdateAll: true}).Create(inst).Error
}
```
