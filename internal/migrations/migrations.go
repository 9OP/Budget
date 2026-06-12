// Package migrations embeds the goose SQL migration files.
package migrations

import "embed"

// FS contains all goose migration files embedded at compile time.
//
//go:embed *.sql
var FS embed.FS
