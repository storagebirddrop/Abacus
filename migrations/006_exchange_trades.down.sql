-- Reverse migration 006

PRAGMA foreign_keys = OFF;

-- Reverse ledger_entries (drop exchange_trade_id, restore NOT NULL on transaction_id)
-- Exchange-sourced entries cannot be reversed; they are dropped.
CREATE TABLE ledger_entries_old (
    id                TEXT PRIMARY KEY,
    wallet_id         TEXT NOT NULL REFERENCES wallets(id),
    transaction_id    TEXT NOT NULL REFERENCES transactions(id),
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

INSERT INTO ledger_entries_old
    (id, wallet_id, transaction_id, type, sats, fiat_amount, fiat_currency,
     price_snapshot_id, category, counterparty_id, note, created_at)
SELECT id, wallet_id, transaction_id, type, sats, fiat_amount, fiat_currency,
       price_snapshot_id, category, counterparty_id, note, created_at
FROM ledger_entries
WHERE transaction_id IS NOT NULL;

DROP TABLE ledger_entries;
ALTER TABLE ledger_entries_old RENAME TO ledger_entries;
CREATE INDEX idx_ledger_wallet_id ON ledger_entries(wallet_id);
CREATE INDEX idx_ledger_transaction_id ON ledger_entries(transaction_id);

-- Drop exchange_trades
DROP TABLE exchange_trades;

-- Reverse wallets source CHECK (exchange wallets are dropped)
CREATE TABLE wallets_old (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    descriptor  TEXT NOT NULL,
    fingerprint TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL CHECK (type IN ('singlesig', 'multisig')),
    network     TEXT NOT NULL CHECK (network IN ('mainnet', 'testnet', 'signet')),
    source      TEXT NOT NULL CHECK (source IN ('sparrow', 'nunchuk', 'bsms', 'manual')),
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
INSERT INTO wallets_old SELECT * FROM wallets WHERE source != 'exchange';
DROP TABLE wallets;
ALTER TABLE wallets_old RENAME TO wallets;

PRAGMA foreign_keys = ON;
