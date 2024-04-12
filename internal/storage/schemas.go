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
	"time"

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

		// delete legacy migration database after 01/10/2024
		if time.Now().After(time.Date(2024, 10, 1, 0, 0, 0, 0, time.Local)) {
			if _, err := db.Exec(`DROP TABLE IF EXISTS ` + tenant("darwin_migrations")); err != nil {
				return err
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

		schemaID := strings.TrimRight(s.Name(), ".sql")

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

		err = t1.Execute(buf, nil)

		if _, err := db.Exec(buf.String()); err != nil {
			return err
		}

		if _, err := db.Exec(`INSERT INTO `+tenant("schemas")+` (Version) VALUES (?)`, s.Semver); err != nil {
			return err
		}

		logger.Log().Debugf("[db] applied schema: %s", s.Name)
	}

	return nil
}

// These functions are used to migrate data formats/structure on startup.
func dataMigrations() {
	// ensure DeletedSize has a value if empty
	if SettingGet("DeletedSize") == "" {
		_ = SettingPut("DeletedSize", "0")
	}

	migrateTagsToManyMany()
}

// Migrate tags to ManyMany structure
// Migration task implemented 12/2023
// TODO: Can be removed end 06/2024 and Tags column & index dropped from mailbox
func migrateTagsToManyMany() {
	toConvert := make(map[string][]string)
	q := sqlf.
		Select("ID, Tags").
		From(tenant("mailbox")).
		Where("Tags != ?", "[]").
		Where("Tags IS NOT NULL")

	if err := q.QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
		var id string
		var jsonTags string
		if err := row.Scan(&id, &jsonTags); err != nil {
			logger.Log().Errorf("[migration] %s", err.Error())
			return
		}

		tags := []string{}

		if err := json.Unmarshal([]byte(jsonTags), &tags); err != nil {
			logger.Log().Errorf("[json] %s", err.Error())
			return
		}

		toConvert[id] = tags
	}); err != nil {
		logger.Log().Errorf("[migration] %s", err.Error())
	}

	if len(toConvert) > 0 {
		logger.Log().Infof("[migration] converting %d message tags", len(toConvert))
		for id, tags := range toConvert {
			if err := SetMessageTags(id, tags); err != nil {
				logger.Log().Errorf("[migration] %s", err.Error())
			} else {
				if _, err := sqlf.Update(tenant("mailbox")).
					Set("Tags", nil).
					Where("ID = ?", id).
					ExecAndClose(context.TODO(), db); err != nil {
					logger.Log().Errorf("[migration] %s", err.Error())
				}
			}
		}

		logger.Log().Info("[migration] tags conversion complete")
	}

	// set all legacy `[]` tags to NULL
	if _, err := sqlf.Update(tenant("mailbox")).
		Set("Tags", nil).
		Where("Tags = ?", "[]").
		ExecAndClose(context.TODO(), db); err != nil {
		logger.Log().Errorf("[migration] %s", err.Error())
	}
}
