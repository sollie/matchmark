package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/sollie/matchmark/internal/rulebook"
)

type EntryIdentity struct {
	Club           string
	Country        string
	Classification *int
	Class          *int
}

func (e Entry) Frozen() bool { return e.Club != nil }

func (e Entry) Identity(club, country string, classes map[string]int) EntryIdentity {
	if e.Frozen() {
		id := EntryIdentity{Club: *e.Club, Class: e.Class, Classification: e.ClassificationAtMatch}
		if e.Country != nil {
			id.Country = *e.Country
		}
		return id
	}
	id := EntryIdentity{Club: club, Country: country}
	if v, ok := classes[e.Category]; ok {
		id.Classification = &v
	}
	return id
}

func classificationAtMatchExpr() string {
	var b strings.Builder
	b.WriteString("CASE category")
	for _, c := range rulebook.ClassifiedCategories() {
		b.WriteString(" WHEN '" + c.Name + "' THEN (SELECT " + c.Column + " FROM shooters WHERE shooters.id = entries.shooter_id)")
	}
	b.WriteString(" END")
	return b.String()
}

var freezeSQL = `UPDATE entries SET
	club = (SELECT club FROM shooters WHERE shooters.id = entries.shooter_id),
	country = (SELECT country FROM shooters WHERE shooters.id = entries.shooter_id),
	classification_at_match = ` + classificationAtMatchExpr() + `
	WHERE match_id = ? AND (? = 0 OR id = ?)`

func classFromClassificationSQL() string {
	var b strings.Builder
	b.WriteString("category IN (")
	for i, c := range rulebook.ClassifiedCategories() {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("'" + c.Name + "'")
	}
	b.WriteString(")")
	return `UPDATE entries SET class = CASE WHEN classification_at_match = 0 OR (classification_at_match IS NULL AND ` + b.String() + `) THEN ? ELSE classification_at_match END
	WHERE match_id = ? AND (? = 0 OR id = ?)`
}

func freezeEntries(ctx context.Context, tx *sql.Tx, matchID, entryID int64, unclassifiedClass int) error {
	if _, err := tx.ExecContext(ctx, freezeSQL, matchID, entryID, entryID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, classFromClassificationSQL(), unclassifiedClass, matchID, entryID, entryID)
	return err
}

func (r *Repository) SetEntryClassification(ctx context.Context, matchID, entryID int64, classification, class int) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE entries SET classification_at_match = ?, class = ? WHERE id = ? AND match_id = ? AND club IS NOT NULL`,
		classification, class, entryID, matchID)
	return conflictIfNone(res, err)
}
