// Package storage handles all database actions
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/klauspost/compress/zstd"
	"github.com/leporo/sqlf"

	// sqlite - https://gitlab.com/cznic/sqlite
	_ "modernc.org/sqlite"

	// rqlite - https://github.com/rqlite/gorqlite | https://rqlite.io/
	_ "github.com/rqlite/gorqlite/stdlib"
)

var (
	db           *sql.DB
	dbFile       string
	dbIsTemp     bool
	sqlDriver    string
	dbLastAction time.Time

	// zstd compression encoder & decoder
	dbEncoder, _ = zstd.NewWriter(nil)
	dbDecoder, _ = zstd.NewReader(nil)
)

// InitDB will initialise the database
func InitDB() error {
	p := config.Database
	var dsn string

	if p == "" {
		// when no path is provided then we create a temporary file
		// which will get deleted on Close(), SIGINT or SIGTERM
		p = fmt.Sprintf("%s-%d.db", path.Join(os.TempDir(), "mailpit"), time.Now().UnixNano())
		dbIsTemp = true
		sqlDriver = "sqlite"
		dsn = p
		logger.Log().Debugf("[db] using temporary database: %s", p)
	} else if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		sqlDriver = "rqlite"
		dsn = p
		logger.Log().Debugf("[db] opening rqlite database %s", p)
	} else {
		p = filepath.Clean(p)
		sqlDriver = "sqlite"
		dsn = fmt.Sprintf("file:%s?cache=shared", p)
		logger.Log().Debugf("[db] opening database %s", p)
	}

	config.Database = p

	if sqlDriver == "sqlite" {
		if !isFile(p) {
			// try create a file to ensure permissions
			f, err := os.Create(p)
			if err != nil {
				return fmt.Errorf("[db] %s", err.Error())
			}
			_ = f.Close()
		}
	}

	var err error

	db, err = sql.Open(sqlDriver, dsn)
	if err != nil {
		return err
	}

	for i := 1; i < 6; i++ {
		if err := Ping(); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			logger.Log().Infof("[db] reconnecting in 5 seconds (attempt %d/5)", i)
			time.Sleep(5 * time.Second)
		} else {
			continue
		}
	}

	// prevent "database locked" errors
	// @see https://github.com/mattn/go-sqlite3#faq
	db.SetMaxOpenConns(1)

	if sqlDriver == "sqlite" {
		// SQLite performance tuning (https://phiresky.github.io/blog/2020/sqlite-performance-tuning/)
		_, err = db.Exec("PRAGMA journal_mode = WAL; PRAGMA synchronous = normal;")
		if err != nil {
			return err
		}
	}

	// create tables if necessary & apply migrations
	if err := dbApplySchemas(); err != nil {
		return err
	}

	LoadTagFilters()

	dbFile = p
	dbLastAction = time.Now()

	sigs := make(chan os.Signal, 1)
	// catch all signals since not explicitly listing
	// Program that will listen to the SIGINT and SIGTERM
	// SIGINT will listen to CTRL-C.
	// SIGTERM will be caught if kill command executed
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	// method invoked upon seeing signal
	go func() {
		s := <-sigs
		fmt.Printf("[db] got %s signal, shutting down\n", s)
		Close()
		os.Exit(0)
	}()

	// auto-prune & delete
	go dbCron()

	go dataMigrations()

	return nil
}

// Tenant applies an optional prefix to the table name
func tenant(table string) string {
	return fmt.Sprintf("%s%s", config.TenantID, table)
}

// Close will close the database, and delete if temporary
func Close() {
	// on a fatal exit (eg: ports blocked), allow Mailpit to run migration tasks before closing the DB
	time.Sleep(200 * time.Millisecond)

	if db != nil {
		if err := db.Close(); err != nil {
			logger.Log().Warn("[db] error closing database, ignoring")
		}
	}

	// allow SQLite to finish closing DB & write WAL logs if local
	time.Sleep(100 * time.Millisecond)

	if dbIsTemp && isFile(dbFile) {
		logger.Log().Debugf("[db] deleting temporary file %s", dbFile)
		if err := os.Remove(dbFile); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
		}
	}
}

// Ping the database connection and return an error if unsuccessful
func Ping() error {
	return db.Ping()
}

// StatsGet returns the total/unread statistics for a mailbox
func StatsGet() MailboxStats {
	var (
		total  = CountTotal()
		unread = CountUnread()
		tags   = GetAllTags()
	)

	dbLastAction = time.Now()

	return MailboxStats{
		Total:  total,
		Unread: unread,
		Tags:   tags,
	}
}

// CountTotal returns the number of emails in the database
func CountTotal() float64 {
	var total float64

	_ = sqlf.From(tenant("mailbox")).
		Select("COUNT(*)").To(&total).
		QueryRowAndClose(context.TODO(), db)

	return total
}

// CountUnread returns the number of emails in the database that are unread.
func CountUnread() float64 {
	var total float64

	_ = sqlf.From(tenant("mailbox")).
		Select("COUNT(*)").To(&total).
		Where("Read = ?", 0).
		QueryRowAndClose(context.TODO(), db)

	return total
}

// CountRead returns the number of emails in the database that are read.
func CountRead() float64 {
	var total float64

	_ = sqlf.From(tenant("mailbox")).
		Select("COUNT(*)").To(&total).
		Where("Read = ?", 1).
		QueryRowAndClose(context.TODO(), db)

	return total
}

// DbSize returns the size of the SQLite database.
func DbSize() float64 {
	var total sql.NullFloat64

	err := db.QueryRow("SELECT page_count * page_size AS size FROM pragma_page_count(), pragma_page_size()").Scan(&total)

	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return total.Float64
	}

	return total.Float64
}

// IsUnread returns whether a message is unread or not.
func IsUnread(id string) bool {
	var unread int

	_ = sqlf.From(tenant("mailbox")).
		Select("COUNT(*)").To(&unread).
		Where("Read = ?", 0).
		Where("ID = ?", id).
		QueryRowAndClose(context.TODO(), db)

	return unread == 1
}

// MessageIDExists checks whether a Message-ID exists in the DB
func MessageIDExists(id string) bool {
	var total int

	_ = sqlf.From(tenant("mailbox")).
		Select("COUNT(*)").To(&total).
		Where("MessageID = ?", id).
		QueryRowAndClose(context.TODO(), db)

	return total != 0
}
