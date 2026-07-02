-- Migration 006: exchange trade support
-- Adds exchange_trades table, expands wallets.source CHECK to include 'exchange',
-- and makes ledger_entries.transaction_id nullable (adds exchange_trade_id FK).
-- SQLite requires table recreation for CHECK constraint and NOT NULL changes.

PRAGMA foreign_keys = OFF;

-- 1. Expand wallets.source CHECK to include 'exchange'
CREATE TABLE wallets_new (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    descriptor  TEXT NOT NULL,
    fingerprint TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL CHECK (type IN ('singlesig', 'multisig')),
    network     TEXT NOT NULL CHECK (network IN ('mainnet', 'testnet', 'signet')),
    source      TEXT NOT NULL CHECK (source IN ('sparrow', 'nunchuk', 'bsms', 'manual', 'exchange')),
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
INSERT INTO wallets_new SELECT * FROM wallets;
DROP TABLE wallets;
ALTER TABLE wallets_new RENAME TO wallets;

-- 2. Create exchange_trades table
CREATE TABLE exchange_trades (
    id            TEXT PRIMARY KEY,
    wallet_id     TEXT NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    trade_type    TEXT NOT NULL CHECK (trade_type IN ('buy','sell','fee','deposit','withdrawal','lightning_receive','lightning_send')),
    exchange      TEXT NOT NULL,
    external_id   TEXT,
    traded_at     INTEGER NOT NULL,
    sats          INTEGER NOT NULL,
    fiat_amount   INTEGER NOT NULL,
    fiat_currency TEXT NOT NULL,
    fee_sats      INTEGER NOT NULL DEFAULT 0,
    fee_fiat      INTEGER NOT NULL DEFAULT 0,
    note          TEXT NOT NULL DEFAULT '',
    created_at    INTEGER NOT NULL
);

CREATE INDEX idx_exchange_trades_wallet ON exchange_trades(wallet_id, traded_at);
CREATE UNIQUE INDEX idx_exchange_trades_external ON exchange_trades(wallet_id, external_id) WHERE external_id IS NOT NULL;

-- 3. Recreate ledger_entries: make transaction_id nullable, add exchange_trade_id
CREATE TABLE ledger_entries_new (
    id                TEXT PRIMARY KEY,
    wallet_id         TEXT NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    transaction_id    TEXT REFERENCES transactions(id) ON DELETE CASCADE,
    exchange_trade_id TEXT REFERENCES exchange_trades(id) ON DELETE CASCADE,
    type              TEXT NOT NULL CHECK (type IN ('debit', 'credit')),
    sats              INTEGER NOT NULL DEFAULT 0,
    fiat_amount       INTEGER NOT NULL DEFAULT 0,
    fiat_currency     TEXT NOT NULL DEFAULT '',
    price_snapshot_id TEXT,
    category          TEXT NOT NULL DEFAULT 'unknown',
    counterparty_id   TEXT,
    note              TEXT NOT NULL DEFAULT '',
    created_at        INTEGER NOT NULL
);

INSERT INTO ledger_entries_new
    (id, wallet_id, transaction_id, exchange_trade_id, type, sats,
     fiat_amount, fiat_currency, price_snapshot_id, category, counterparty_id, note, created_at)
SELECT id, wallet_id, transaction_id, NULL, type, sats,
       fiat_amount, fiat_currency, price_snapshot_id, category, counterparty_id, note, created_at
FROM ledger_entries;

DROP TABLE ledger_entries;
ALTER TABLE ledger_entries_new RENAME TO ledger_entries;

CREATE INDEX idx_ledger_wallet_id ON ledger_entries(wallet_id);
CREATE INDEX idx_ledger_transaction_id ON ledger_entries(transaction_id) WHERE transaction_id IS NOT NULL;
CREATE INDEX idx_ledger_exchange_trade_id ON ledger_entries(exchange_trade_id) WHERE exchange_trade_id IS NOT NULL;

PRAGMA foreign_keys = ON;
