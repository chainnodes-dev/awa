// Package migrations exposes the embedded SQL migration files.
// The FS is consumed by internal/migrate via iofs.New(migrations.FS, ".").
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
