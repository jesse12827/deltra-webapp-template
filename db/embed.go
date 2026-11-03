// Package db embeds the SQL files that the application binary needs at run
// time. It is one of only two Go files outside cmd/ and internal/, because Go's
// embed feature can include only files at or below the embedding file's directory
// (Web-App Development Stack v2.0, Section 6).
package db

import "embed"

// Migrations holds db/migrations/*.sql, applied by `<app> migrate`.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Bootstrap holds db/bootstrap/*.sql, used by the test harness to build
// disposable databases exactly as the Approver builds production ones.
//
//go:embed bootstrap/*.sql
var Bootstrap embed.FS
