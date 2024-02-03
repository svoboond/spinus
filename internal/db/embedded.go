package db

import "embed"

//go:embed migration
var EmbeddedContentMigration embed.FS
