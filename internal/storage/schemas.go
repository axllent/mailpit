package storage

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"log"
	"path"
	"sort"
	"strings"
	"text/template"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/semver"
	"github.com/leporo/sqlf"
)

//go:embed schemas/*
var schemaScripts embed.FS

// Create tables and apply schemas if required
func dbApplySchemas() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + tenant("schemas") + ` (Version TEXT PRIMARY KEY NOT NULL)`); err != nil {
		return err
	}

	var legacyMigrationTable int
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)`, tenant("darwin_migrations")).Scan(&legacyMigrationTable)
	if err != nil {
		return err
	}

	if legacyMigrationTable == 1 {
		rows, err := db.Query(`SELECT version FROM ` + tenant("darwin_migrations"))
		if err != nil {
			return err
		}

		legacySchemas := []string{}

		for rows.Next() {
			var oldID string
			if err := rows.Scan(&oldID); err == nil {
				legacySchemas = append(legacySchemas, semver.MajorMinor(oldID)+"."+semver.Patch(oldID))
			}
		}

		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}

		legacySchemas = semver.SortMin(legacySchemas)

		for _, v := range legacySchemas {
			var migrated int
			err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM `+tenant("schemas")+` WHERE Version = ?)`, v).Scan(&migrated)
			if err != nil {
				return err
			}
			if migrated == 0 {
				// copy to tenant("schemas")
				if _, err := db.Exec(`INSERT INTO `+tenant("schemas")+` (Version) VALUES (?)`, v); err != nil {
					return err
				}
			}
		}
	}

	schemaFiles, err := schemaScripts.ReadDir("schemas")
	if err != nil {
		log.Fatal(err)
	}

	temp := template.New("")
	temp.Funcs(
		template.FuncMap{
			"tenant": tenant,
		},
	)

	type schema struct {
		Name   string
		Semver string
	}

	scripts := []schema{}

	for _, s := range schemaFiles {
		if !s.Type().IsRegular() || !strings.HasSuffix(s.Name(), ".sql") {
			continue
		}

		schemaID := strings.TrimSuffix(s.Name(), ".sql")

		if !semver.IsValid(schemaID) {
			logger.Log().Warnf("[db] invalid schema name: %s", s.Name())
			continue
		}

		script := schema{s.Name(), semver.MajorMinor(schemaID) + "." + semver.Patch(schemaID)}
		scripts = append(scripts, script)
	}

	// sort schemas by semver, low to high
	sort.Slice(scripts, func(i, j int) bool {
		return semver.Compare(scripts[j].Semver, scripts[i].Semver) == 1
	})

	// detect whether this is an existing database (has prior schema entries)
	var existingSchemaCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + tenant("schemas")).Scan(&existingSchemaCount); err != nil {
		return err
	}
	isExistingDB := existingSchemaCount > 0

	for _, s := range scripts {
		var complete int
		err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM `+tenant("schemas")+` WHERE Version = ?)`, s.Semver).Scan(&complete)
		if err != nil {
			return err
		}

		if complete == 1 {
			// already completed, ignore
			continue
		}

		if isExistingDB {
			logger.Log().Infof("[db] applying schema updates: %s", s.Name)
		} else {
			logger.Log().Debugf("[db] applying schema updates: %s", s.Name)
		}

		// use path.Join for Windows compatibility, see https://github.com/golang/go/issues/44305
		b, err := schemaScripts.ReadFile(path.Join("schemas", s.Name))
		if err != nil {
			return err
		}

		// parse import script
		t1, err := temp.Parse(string(b))
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)

		if err := t1.Execute(buf, nil); err != nil {
			return err
		}

		if _, err := db.Exec(buf.String()); err != nil {
			return err
		}

		if _, err := db.Exec(`INSERT INTO `+tenant("schemas")+` (Version) VALUES (?)`, s.Semver); err != nil {
			return err
		}
	}

	return nil
}

// These functions are used to migrate data formats/structure on startup.
func dataMigrations() {
	// ensure DeletedSize has a value if empty
	if SettingGet("DeletedSize") == "" {
		_ = SettingPut("DeletedSize", "0")
	}

	if err := backfillMetaSearchText(); err != nil {
		logger.Log().Errorf("[db] populating MetaSearchText: %s", err.Error())
	}

	if err := refoldSearchText(); err != nil {
		logger.Log().Errorf("[db] case-folding SearchText: %s", err.Error())
	}
}

// searchTextFoldedSetting is set once refoldSearchText has processed every message
const searchTextFoldedSetting = "SearchTextFolded"

// RefoldSearchText applies foldCase to the SearchText of messages stored before search
// text was case-folded (v1.34.0), when it was only lowercased. Lowercasing is idempotent
// so foldCase(storedText) is exactly what the message would now be indexed as, and no
// messages need to be parsed. Only messages containing characters which case folding
// changes (eg: `ß`, `ς`) are updated. It runs once, and is safe to re-run if interrupted.
// Until a message is updated, general searches for words containing those characters
// may not match it.
func refoldSearchText() error {
	if SettingGet(searchTextFoldedSetting) == "1" {
		return nil
	}

	type update struct {
		ID         string
		SearchText string
		Old        string
	}

	chunkSize := 1000
	lastID := ""
	checked := 0
	updated := 0
	logged := false

	for {
		updates := []update{}
		var id, searchText string
		rows := 0

		err := sqlf.From(tenant("mailbox")).
			Select("ID").To(&id).
			Select("IFNULL(SearchText, '')").To(&searchText).
			Where("ID > ?", lastID).
			OrderBy("ID").
			Limit(chunkSize).
			QueryAndClose(context.TODO(), db, func(_ *sql.Rows) {
				rows++
				lastID = id

				if folded := foldCase(searchText); folded != searchText {
					updates = append(updates, update{id, folded, searchText})
				}
			})
		if err != nil {
			return err
		}

		if rows == 0 {
			break
		}

		checked += rows

		if !logged && checked >= chunkSize {
			logger.Log().Info("[db] case-folding search text for existing messages")
			logged = true
		}

		if len(updates) == 0 {
			continue
		}

		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			return err
		}

		for _, u := range updates {
			// a message reindexed in the meantime already has the current value
			if _, err := tx.Exec(`UPDATE `+tenant("mailbox")+` SET SearchText = ? WHERE ID = ? AND SearchText = ?`, u.SearchText, u.ID, u.Old); err != nil {
				_ = tx.Rollback()
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		updated += len(updates)
	}

	if logged || updated > 0 {
		logger.Log().Infof("[db] case-folded search text: checked %d messages, updated %d", checked, updated)
	}

	return SettingPut(searchTextFoldedSetting, "1")
}

// BackfillMetaSearchText populates the MetaSearchText column for messages stored before
// it existed (schema 1.34.0). It uses the stored metadata & subject, so no messages
// need to be parsed. Until a message is populated, field searches fall back to the
// original columns, which are only case-insensitive for ASCII characters.
func backfillMetaSearchText() error {
	var pending float64 // use float64 for rqlite compatibility
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + tenant("mailbox") + ` WHERE MetaSearchText IS NULL`).Scan(&pending); err != nil {
		return err
	}

	if pending == 0 {
		return nil
	}

	logger.Log().Infof("[db] populating MetaSearchText for %d messages", int(pending))

	type update struct {
		ID             string
		MetaSearchText string
	}

	chunkSize := 1000
	lastID := ""
	done := 0
	failed := 0

	for {
		updates := []update{}
		var id, metadata, subject string
		rows := 0

		// walk by ID rather than re-selecting `MetaSearchText IS NULL`, so a message
		// that cannot be processed is skipped instead of being selected forever
		err := sqlf.From(tenant("mailbox")).
			Select("ID").To(&id).
			Select("IFNULL(Metadata, '{}')").To(&metadata).
			Select("IFNULL(Subject, '')").To(&subject).
			Where("MetaSearchText IS NULL AND ID > ?", lastID).
			OrderBy("ID").
			Limit(chunkSize).
			QueryAndClose(context.TODO(), db, func(_ *sql.Rows) {
				rows++
				lastID = id

				var meta Metadata
				if err := json.Unmarshal([]byte(metadata), &meta); err != nil {
					logger.Log().Warnf("[db] cannot populate MetaSearchText for message %s: %s", id, err.Error())
					failed++
					return
				}

				mst, err := createMetaSearchText(meta, subject)
				if err != nil {
					logger.Log().Warnf("[db] cannot populate MetaSearchText for message %s: %s", id, err.Error())
					failed++
					return
				}

				updates = append(updates, update{id, mst})
			})
		if err != nil {
			return err
		}

		if rows == 0 {
			break
		}

		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			return err
		}

		for _, u := range updates {
			// a message stored or reindexed in the meantime already has the current value
			if _, err := tx.Exec(`UPDATE `+tenant("mailbox")+` SET MetaSearchText = ? WHERE ID = ? AND MetaSearchText IS NULL`, u.MetaSearchText, u.ID); err != nil {
				_ = tx.Rollback()
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		done += len(updates)
	}

	if failed > 0 {
		logger.Log().Warnf("[db] populated MetaSearchText for %d messages, %d failed", done, failed)
	} else {
		logger.Log().Infof("[db] populated MetaSearchText for %d messages", done)
	}

	return nil
}
