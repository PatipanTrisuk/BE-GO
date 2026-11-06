package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"go-be/config"
)

func main() {
	_ = godotenv.Load()

	db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Fatalf("ensure schema_migrations: %v", err)
	}

	// Bootstrap: if users already exist, mark older migrations as applied
	// so only new files (like soft delete) run.
	var usersExist bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'users'
		)
	`).Scan(&usersExist); err != nil {
		log.Fatalf("check users table: %v", err)
	}

	entries, err := os.ReadDir("migrations")
	if err != nil {
		log.Fatalf("read migrations: %v", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	if usersExist {
		for _, name := range files {
			if name >= "008_" {
				break
			}
			if _, err := db.Exec(ctx, `
				INSERT INTO schema_migrations (filename)
				VALUES ($1)
				ON CONFLICT (filename) DO NOTHING
			`, name); err != nil {
				log.Fatalf("bootstrap %s: %v", name, err)
			}
		}
	}

	for _, name := range files {
		var exists bool
		if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`, name).Scan(&exists); err != nil {
			log.Fatalf("check migration %s: %v", name, err)
		}
		if exists {
			fmt.Printf("skip %s\n", name)
			continue
		}

		sqlBytes, err := os.ReadFile(filepath.Join("migrations", name))
		if err != nil {
			log.Fatalf("read %s: %v", name, err)
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			log.Fatalf("begin %s: %v", name, err)
		}

		for _, stmt := range splitSQL(string(sqlBytes)) {
			if _, err := tx.Exec(ctx, stmt); err != nil {
				_ = tx.Rollback(ctx)
				log.Fatalf("apply %s: %v\nstatement: %s", name, err, stmt)
			}
		}

		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("record %s: %v", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			log.Fatalf("commit %s: %v", name, err)
		}
		fmt.Printf("applied %s\n", name)
	}
}

func splitSQL(raw string) []string {
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		out = append(out, stmt)
	}
	return out
}
