package data

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// dialectorName returns the canonical driver name for the dialector
// attached to the given *gorm.DB. Mirrors the helper in the billing
// data layer; the two are kept package-local to avoid a cross-domain
// import.
func dialectorName(db *gorm.DB) string {
	if db == nil {
		return ""
	}
	if db.Dialector == nil {
		return ""
	}
	return db.Dialector.Name()
}

// forUpdateClause returns the dialect-appropriate SELECT ... FOR UPDATE
// clause.
func forUpdateClause(driver string) clause.Locking {
	if driver == "sqlite3" {
		return clause.Locking{}
	}
	return clause.Locking{Strength: "UPDATE"}
}
