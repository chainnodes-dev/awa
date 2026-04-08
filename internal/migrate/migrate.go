// Package migrate wraps golang-migrate to apply SQL migrations embedded in
// the binary via the migrations package. Migrations are compiled in at build
// time, so no filesystem path is required at runtime — the same binary works
// in Docker, Kubernetes, and local development without any extra mounts.
package migrate

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/asm-platform/asm/migrations"
)

// Up applies all pending migrations against the given Postgres DSN.
// It is idempotent: calling it on an already-up-to-date database is a no-op.
func Up(databaseURL string) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			slog.Warn("Migration close source error", "error", srcErr)
		}
		if dbErr != nil {
			slog.Warn("Migration close db error", "error", dbErr)
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	v, _, _ := m.Version()
	slog.Info("Migration complete", "schema_version", v)
	return nil
}
