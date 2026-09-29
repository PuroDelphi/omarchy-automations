package core

import (
	"context"
	"database/sql"
	"fmt"
)

const DatabaseVersion = 5
const baseSchema = `CREATE TABLE IF NOT EXISTS schema_version(version INTEGER PRIMARY KEY);

 CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS audit(id INTEGER PRIMARY KEY,at TEXT NOT NULL,operation TEXT NOT NULL,resource TEXT NOT NULL,result TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS revisions(hash TEXT PRIMARY KEY,config TEXT NOT NULL,created_at TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS grants(scope TEXT PRIMARY KEY,hash TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS secrets(id TEXT PRIMARY KEY,backend TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS events(id TEXT NOT NULL,source TEXT NOT NULL,payload TEXT NOT NULL,created_at TEXT NOT NULL,PRIMARY KEY(id,source));
 CREATE TABLE IF NOT EXISTS executions(id TEXT PRIMARY KEY,event_id TEXT NOT NULL,event_source TEXT NOT NULL,revision TEXT NOT NULL,flow TEXT NOT NULL,state TEXT NOT NULL,step INTEGER NOT NULL,created_at TEXT NOT NULL,FOREIGN KEY(event_id,event_source) REFERENCES events(id,source),FOREIGN KEY(revision) REFERENCES revisions(hash));
 CREATE TABLE IF NOT EXISTS steps(execution TEXT NOT NULL,step INTEGER NOT NULL,state TEXT NOT NULL,message TEXT NOT NULL,finished_at TEXT NOT NULL,PRIMARY KEY(execution,step));
 CREATE TABLE IF NOT EXISTS outbox(execution TEXT NOT NULL,step INTEGER NOT NULL,body TEXT NOT NULL,attempts INTEGER NOT NULL,next_at INTEGER NOT NULL,created_at INTEGER NOT NULL,state TEXT NOT NULL,last_status INTEGER,PRIMARY KEY(execution,step));
 CREATE TABLE IF NOT EXISTS monitor_state(id TEXT PRIMARY KEY,hash TEXT NOT NULL,phase TEXT NOT NULL,since INTEGER NOT NULL,last INTEGER NOT NULL,next INTEGER NOT NULL,value REAL NOT NULL,error TEXT NOT NULL);`

func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000; PRAGMA max_page_count=65536;"); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_version(version INTEGER PRIMARY KEY)"); err != nil {
		return err
	}
	var version int
	if err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(version),0) FROM schema_version").Scan(&version); err != nil {
		return err
	}
	if version > DatabaseVersion {
		return fmt.Errorf("database version %d requires a newer Omarchy Automations; no migration applied", version)
	}
	// Version 1 development builds created tables incrementally. Idempotent DDL
	// preserves their rows while bringing that baseline up to its full contract.
	if _, err = tx.ExecContext(ctx, baseSchema); err != nil {
		return err
	}
	if version < 2 {
		if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS seen_deliveries(id TEXT NOT NULL,source TEXT NOT NULL,seen_at INTEGER NOT NULL,PRIMARY KEY(id,source));
   INSERT OR IGNORE INTO seen_deliveries(id,source,seen_at) SELECT id,source,CAST(strftime('%s',created_at) AS INTEGER) FROM events;
   CREATE INDEX IF NOT EXISTS executions_ready ON executions(state,created_at);
   CREATE INDEX IF NOT EXISTS executions_event ON executions(event_id,event_source);
   CREATE INDEX IF NOT EXISTS events_created ON events(created_at);
   DELETE FROM schema_version;INSERT INTO schema_version VALUES(2);`); err != nil {
			return err
		}
	}
	if version < 3 {
		if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS timer_state(id TEXT PRIMARY KEY,hash TEXT NOT NULL,next_at INTEGER NOT NULL,last_key TEXT NOT NULL,last_at INTEGER NOT NULL);
   DELETE FROM schema_version;INSERT INTO schema_version VALUES(3);`); err != nil {
			return err
		}
	}
	if version < 4 {
		if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS journal_state(id TEXT PRIMARY KEY,hash TEXT NOT NULL,cursor TEXT NOT NULL,since_us INTEGER NOT NULL,next_at INTEGER NOT NULL,last_count INTEGER NOT NULL,error TEXT NOT NULL);
   DELETE FROM schema_version;INSERT INTO schema_version VALUES(4);`); err != nil {
			return err
		}
	}
	if version < 5 {
		if _, err = tx.ExecContext(ctx, `ALTER TABLE executions ADD COLUMN context TEXT NOT NULL DEFAULT '';
            DELETE FROM schema_version;INSERT INTO schema_version VALUES(5);`); err != nil {
			return err
		}
	}
	return tx.Commit()
}
