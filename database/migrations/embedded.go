package migrations

import "embed"

//go:embed *.up.sql *.down.sql
var embeddedMigrations embed.FS
