package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type PriceSnapshotRepo struct {
	db *sql.DB
}

func NewPriceSnapshotRepo(db *sql.DB) *PriceSnapshotRepo {
	return &PriceSnapshotRepo{db: db}
}

func (r *PriceSnapshotRepo) Insert(ctx context.Context, p *domain.PriceSnapshot) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO price_snapshots (id, currency, price_fiat, source, timestamp)
		 VALUES (?, ?, ?, ?, ?)`,
		p.ID, p.Currency, p.PriceFiat, p.Source, p.Timestamp.Unix(),
	)
	return err
}

// GetClosest returns the price snapshot nearest in time to t for the given currency.
// Searches both before and after t, returning whichever is closer.
func (r *PriceSnapshotRepo) GetClosest(ctx context.Context, currency string, t time.Time) (*domain.PriceSnapshot, error) {
	ts := t.Unix()
	row := r.db.QueryRowContext(ctx,
		`SELECT id, currency, price_fiat, source, timestamp
		 FROM price_snapshots
		 WHERE currency=?
		 ORDER BY ABS(timestamp - ?) ASC
		 LIMIT 1`,
		currency, ts,
	)
	return scanPriceSnapshot(row)
}

func (r *PriceSnapshotRepo) List(ctx context.Context, currency string, from, to time.Time) ([]*domain.PriceSnapshot, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, currency, price_fiat, source, timestamp
		 FROM price_snapshots
		 WHERE currency=? AND timestamp>=? AND timestamp<=?
		 ORDER BY timestamp ASC`,
		currency, from.Unix(), to.Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snaps []*domain.PriceSnapshot
	for rows.Next() {
		p, err := scanPriceSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snaps = append(snaps, p)
	}
	return snaps, rows.Err()
}

func scanPriceSnapshot(s scanner) (*domain.PriceSnapshot, error) {
	var p domain.PriceSnapshot
	var ts int64
	err := s.Scan(&p.ID, &p.Currency, &p.PriceFiat, &p.Source, &ts)
	if err != nil {
		return nil, err
	}
	p.Timestamp = time.Unix(ts, 0).UTC()
	return &p, nil
}
