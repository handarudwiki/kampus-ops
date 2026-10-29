// Package migrations embeds the SQL migration files (golang-migrate naming:
// NNNN_name.up.sql / NNNN_name.down.sql) so binaries can migrate on startup.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
