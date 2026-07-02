package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/storagebirddrop/abacus/internal/domain"
)

type ExchangeTradeRepo struct {
	db *sql.DB
}

func NewExchangeTradeRepo(db *sql.DB) *ExchangeTradeRepo {
	return &ExchangeTradeRepo{db: db}
}

func (r *ExchangeTradeRepo) Create(ctx context.Context, tx *sql.Tx, t *domain.ExchangeTrade) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	execer := r.db
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO exchange_trades
			 (id, wallet_id, trade_type, exchange, external_id, traded_at,
			  sats, fiat_amount, fiat_currency, fee_sats, fee_fiat, note, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.WalletID, string(t.TradeType), t.Exchange,
			nullString(t.ExternalID), t.TradedAt.Unix(),
			t.Sats, t.FiatAmount, t.FiatCurrency,
			t.FeeSats, t.FeeFiat, t.Note, t.CreatedAt.Unix(),
		)
	} else {
		_, err = execer.ExecContext(ctx,
			`INSERT OR IGNORE INTO exchange_trades
			 (id, wallet_id, trade_type, exchange, external_id, traded_at,
			  sats, fiat_amount, fiat_currency, fee_sats, fee_fiat, note, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.WalletID, string(t.TradeType), t.Exchange,
			nullString(t.ExternalID), t.TradedAt.Unix(),
			t.Sats, t.FiatAmount, t.FiatCurrency,
			t.FeeSats, t.FeeFiat, t.Note, t.CreatedAt.Unix(),
		)
	}
	return err
}

func (r *ExchangeTradeRepo) ListByWallet(ctx context.Context, walletID string) ([]*domain.ExchangeTrade, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, wallet_id, trade_type, exchange, COALESCE(external_id,''), traded_at,
		        sats, fiat_amount, fiat_currency, fee_sats, fee_fiat, note, created_at
		 FROM exchange_trades WHERE wallet_id=?
		 ORDER BY traded_at ASC`,
		walletID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trades []*domain.ExchangeTrade
	for rows.Next() {
		t, err := scanExchangeTrade(rows)
		if err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}
	return trades, rows.Err()
}

func (r *ExchangeTradeRepo) CountByWallet(ctx context.Context, walletID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM exchange_trades WHERE wallet_id=?`, walletID,
	).Scan(&count)
	return count, err
}

func scanExchangeTrade(s scanner) (*domain.ExchangeTrade, error) {
	var t domain.ExchangeTrade
	var tradeType string
	var tradedUnix, createdUnix int64
	err := s.Scan(
		&t.ID, &t.WalletID, &tradeType, &t.Exchange, &t.ExternalID,
		&tradedUnix, &t.Sats, &t.FiatAmount, &t.FiatCurrency,
		&t.FeeSats, &t.FeeFiat, &t.Note, &createdUnix,
	)
	if err != nil {
		return nil, err
	}
	t.TradeType = domain.TradeType(tradeType)
	t.TradedAt = time.Unix(tradedUnix, 0).UTC()
	t.CreatedAt = time.Unix(createdUnix, 0).UTC()
	return &t, nil
}

// isDuplicateError returns true when SQLite reports a UNIQUE constraint violation.
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
