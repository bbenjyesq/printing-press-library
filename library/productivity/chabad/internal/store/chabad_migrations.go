// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.

package store

import (
	"context"
	"fmt"
)

// Occasion is a person's recurring Hebrew-date event (yahrzeit, birthday, or
// anniversary) kept by `chabad-pp-cli occasions`.
type Occasion struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	HebrewYear  int    `json:"hebrew_year"`
	HebrewMonth int    `json:"hebrew_month"`
	HebrewDay   int    `json:"hebrew_day"`
	Gregorian   string `json:"gregorian_date,omitempty"`
	AfterSunset bool   `json:"after_sunset"`
	Notes       string `json:"notes,omitempty"`
	CreatedAt   string `json:"created_at"`
}

const occasionsSchema = `CREATE TABLE IF NOT EXISTS chabad_occasions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	kind TEXT NOT NULL,
	hebrew_year INTEGER NOT NULL,
	hebrew_month INTEGER NOT NULL,
	hebrew_day INTEGER NOT NULL,
	gregorian_date TEXT NOT NULL DEFAULT '',
	after_sunset INTEGER NOT NULL DEFAULT 0,
	notes TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
)`

// EnsureOccasions creates the occasions table on first use.
func (s *Store) EnsureOccasions(ctx context.Context) error {
	if _, err := s.DB().ExecContext(ctx, occasionsSchema); err != nil {
		return fmt.Errorf("creating occasions table: %w", err)
	}
	return nil
}

// AddOccasion inserts an occasion and returns its id.
func (s *Store) AddOccasion(ctx context.Context, o Occasion) (int64, error) {
	if err := s.EnsureOccasions(ctx); err != nil {
		return 0, err
	}
	after := 0
	if o.AfterSunset {
		after = 1
	}
	res, err := s.DB().ExecContext(ctx,
		`INSERT INTO chabad_occasions (name, kind, hebrew_year, hebrew_month, hebrew_day, gregorian_date, after_sunset, notes) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		o.Name, o.Kind, o.HebrewYear, o.HebrewMonth, o.HebrewDay, o.Gregorian, after, o.Notes)
	if err != nil {
		return 0, fmt.Errorf("saving occasion: %w", err)
	}
	return res.LastInsertId()
}

// ListOccasions returns every saved occasion ordered by name.
func (s *Store) ListOccasions(ctx context.Context) ([]Occasion, error) {
	if err := s.EnsureOccasions(ctx); err != nil {
		return nil, err
	}
	rows, err := s.DB().QueryContext(ctx,
		`SELECT id, name, kind, hebrew_year, hebrew_month, hebrew_day, gregorian_date, after_sunset, notes, created_at FROM chabad_occasions ORDER BY name COLLATE NOCASE, id`)
	if err != nil {
		return nil, fmt.Errorf("listing occasions: %w", err)
	}
	out := make([]Occasion, 0)
	for rows.Next() {
		var o Occasion
		var after int
		if err := rows.Scan(&o.ID, &o.Name, &o.Kind, &o.HebrewYear, &o.HebrewMonth, &o.HebrewDay, &o.Gregorian, &after, &o.Notes, &o.CreatedAt); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("reading occasion: %w", err)
		}
		o.AfterSunset = after == 1
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return out, nil
}

// RemoveOccasion deletes an occasion by id and reports whether it existed.
func (s *Store) RemoveOccasion(ctx context.Context, id int64) (bool, error) {
	if err := s.EnsureOccasions(ctx); err != nil {
		return false, err
	}
	res, err := s.DB().ExecContext(ctx, `DELETE FROM chabad_occasions WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("removing occasion: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
