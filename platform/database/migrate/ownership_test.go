package migrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOwnershipFilter_RestrictsApply(t *testing.T) {
	db := openSqlite(t)
	dir := t.TempDir()

	// Three fake migrations + the bootstrap.
	writeMigration(t, dir, "022_create_schema_migrations.sql", `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT);`)
	writeMigration(t, dir, "100_create_widgets.sql", `CREATE TABLE widgets (id INTEGER);`)
	writeMigration(t, dir, "101_create_gadgets.sql", `CREATE TABLE gadgets (id INTEGER);`)
	writeMigration(t, dir, "102_create_admin_only.sql", `CREATE TABLE admin_only (id INTEGER);`)

	// Manifest: identity owns 100 only.
	manifest := `
shared:
  - 022_create_schema_migrations
services:
  identity:
    - 100_create_widgets
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ownership.yaml"), []byte(manifest), 0o644))

	runner := NewWithDriver(db, dir, "sqlite3").WithOwnershipFilter("identity")
	applied, err := runner.Apply(context.Background())
	require.NoError(t, err)

	// Should apply bootstrap + widgets; NOT gadgets / admin_only.
	assert.ElementsMatch(t, []string{"022_create_schema_migrations", "100_create_widgets"}, applied)

	exists := func(name string) bool { return tableExists(t, db, name) }
	assert.True(t, exists("widgets"), "widgets table should be created")
	assert.False(t, exists("gadgets"), "gadgets should be filtered out")
	assert.False(t, exists("admin_only"), "admin_only should be filtered out")
}

func TestOwnershipFilter_MissingManifestAppliesAll(t *testing.T) {
	db := openSqlite(t)
	dir := t.TempDir()
	writeMigration(t, dir, "022_create_schema_migrations.sql", `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT);`)
	writeMigration(t, dir, "100_create_widgets.sql", `CREATE TABLE widgets (id INTEGER);`)

	// No ownership.yaml — filter is a no-op.
	runner := NewWithDriver(db, dir, "sqlite3").WithOwnershipFilter("identity")
	applied, err := runner.Apply(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"022_create_schema_migrations", "100_create_widgets"}, applied)
}

func TestOwnershipFilter_UnknownServiceAppliesOnlyShared(t *testing.T) {
	db := openSqlite(t)
	dir := t.TempDir()
	writeMigration(t, dir, "022_create_schema_migrations.sql", `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT);`)
	writeMigration(t, dir, "100_create_widgets.sql", `CREATE TABLE widgets (id INTEGER);`)

	manifest := `
shared:
  - 022_create_schema_migrations
services:
  identity:
    - 100_create_widgets
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ownership.yaml"), []byte(manifest), 0o644))

	runner := NewWithDriver(db, dir, "sqlite3").WithOwnershipFilter("unknown-service")
	applied, err := runner.Apply(context.Background())
	require.NoError(t, err)
	// Unknown service → only shared files apply.
	assert.ElementsMatch(t, []string{"022_create_schema_migrations"}, applied)
	assert.False(t, tableExists(t, db, "widgets"))
}
