package migrate

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Migration struct {
	Name string
	Hash string
}

type MigrationCollection struct {
	migrations []Migration
	db         *sql.DB
}

func md5String(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func (mc *MigrationCollection) loadExistingMigrations(direction *string) {
	var dir string
	if direction == nil {
		dir = "asc"
	} else {
		dir = *direction
	}
	result, err := mc.db.Query("select name,hash from migrations order by ?;", dir)
	if err != nil {
		if strings.Contains(err.Error(), "no such table: migrations") {
			ensureMigrationTable(mc.db)
		} else {
			log.Fatalf("Failed to get existing migrations with error: %v", err)
		}
	}

	existing := []Migration{}

	if result != nil {
		for result.Next() {
			x := Migration{}
			if err := result.Scan(&x.Name, &x.Hash); err != nil {
				log.Fatalf("failed to read row with err: %v", err)
			}
			existing = append(existing, x)
		}
	}

	mc.migrations = existing
}

func (mc *MigrationCollection) hasMigration(name, hash string) bool {
	for _, mig := range mc.migrations {
		if mig.Name != name {
			continue
		}
		if mig.Hash == hash {
			return true
		} else {
			log.Fatalf("similar named migration with different hashes received, %v:%v, (savedHash:%v)", name, hash, mig.Hash)
		}
	}
	return false
}

func ensureMigrationTable(db *sql.DB) {
	db.Exec("CREATE TABLE if not exists migrations(id INTEGER PRIMARY KEY AUTOINCREMENT,name TEXT NOT NULL UNIQUE,hash TEXT,created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);")
}

func UpdateUpAudit(db *sql.DB, executed map[string]string) {
	ensureMigrationTable(db)
	tx, _ := db.Begin()
	for name, hash := range executed {
		tx.Exec("insert into migrations (name, hash) values (?,?)", name, hash)
	}
	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

func MigrateUp(db *sql.DB, dir string) {
	mc := &MigrationCollection{
		db: db,
	}
	mc.loadExistingMigrations(nil)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("error reading migrations, %v", err)
	}

	executedFiles := map[string]string{}
	for _, file := range entries {
		if !strings.HasSuffix(file.Name(), ".up.sql") {
			continue
		}
		tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
		if err != nil {
			panic(err)
		}
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			panic(err)
		}

		fileHash := md5String(data)
		if mc.hasMigration(file.Name(), fileHash) {
			continue
		}

		_, err = tx.Exec(string(data))
		if err != nil {
			panic(err)
		}

		if err = tx.Commit(); err != nil {
			panic(err)
		}

		executedFiles[file.Name()] = fileHash
		log.Printf("Ran %v\n", file)
	}
	UpdateUpAudit(db, executedFiles)
}
