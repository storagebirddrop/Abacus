-- Abacus initial schema
-- All monetary values are stored as integer satoshis (never floats).
-- All timestamps are stored as Unix epoch integers.

CREATE TABLE wallets (
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

CREATE TABLE addresses (
    id              TEXT PRIMARY KEY,
    wallet_id       TEXT NOT NULL REFERENCES wallets(id),
    address         TEXT NOT NULL,
    derivation_path TEXT NOT NULL DEFAULT '',
    type            TEXT NOT NULL CHECK (type IN ('receive', 'change')),
    label           TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_addresses_wallet_address ON addresses(wallet_id, address);

CREATE TABLE transactions (
    id           TEXT PRIMARY KEY,
    wallet_id    TEXT NOT NULL REFERENCES wallets(id),
    txid         TEXT NOT NULL,
    block_height INTEGER NOT NULL DEFAULT 0,
    block_hash   TEXT NOT NULL DEFAULT '',
    block_time   INTEGER NOT NULL DEFAULT 0,
    fee_sats     INTEGER NOT NULL DEFAULT 0,
    confirmed    INTEGER NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_transactions_wallet_txid ON transactions(wallet_id, txid);
CREATE INDEX idx_transactions_txid ON transactions(txid);
CREATE INDEX idx_transactions_block_height ON transactions(block_height);

CREATE TABLE transaction_inputs (
    id             TEXT PRIMARY KEY,
    transaction_id TEXT NOT NULL REFERENCES transactions(id),
    prev_txid      TEXT NOT NULL,
    prev_vout      INTEGER NOT NULL,
    sats           INTEGER NOT NULL DEFAULT 0,
    address        TEXT NOT NULL DEFAULT '',
    sequence       INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_tx_inputs_transaction_id ON transaction_inputs(transaction_id);

CREATE TABLE transaction_outputs (
    id             TEXT PRIMARY KEY,
    transaction_id TEXT NOT NULL REFERENCES transactions(id),
    vout           INTEGER NOT NULL,
    sats           INTEGER NOT NULL DEFAULT 0,
    address        TEXT NOT NULL DEFAULT '',
    script_pubkey  TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_tx_outputs_transaction_id ON transaction_outputs(transaction_id);

CREATE TABLE utxos (
    id           TEXT PRIMARY KEY,
    wallet_id    TEXT NOT NULL REFERENCES wallets(id),
    txid         TEXT NOT NULL,
    vout         INTEGER NOT NULL,
    sats         INTEGER NOT NULL DEFAULT 0,
    address      TEXT NOT NULL DEFAULT '',
    block_height INTEGER NOT NULL DEFAULT 0,
    block_time   INTEGER NOT NULL DEFAULT 0,
    spent        INTEGER NOT NULL DEFAULT 0,
    spent_txid   TEXT NOT NULL DEFAULT '',
    label        TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX idx_utxos_txid_vout ON utxos(txid, vout);
CREATE INDEX idx_utxos_wallet_id ON utxos(wallet_id);
CREATE INDEX idx_utxos_spent ON utxos(spent);

-- ledger_entries is append-only. Updates are forbidden; use journal_entries.
CREATE TABLE ledger_entries (
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

CREATE INDEX idx_ledger_wallet_id ON ledger_entries(wallet_id);
CREATE INDEX idx_ledger_transaction_id ON ledger_entries(transaction_id);

CREATE TABLE journal_entries (
    id              TEXT PRIMARY KEY,
    ledger_entry_id TEXT NOT NULL REFERENCES ledger_entries(id),
    field_changed   TEXT NOT NULL,
    old_value       TEXT NOT NULL DEFAULT '',
    new_value       TEXT NOT NULL DEFAULT '',
    reason          TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL
);

CREATE INDEX idx_journal_ledger_entry_id ON journal_entries(ledger_entry_id);

CREATE TABLE cost_basis_records (
    id            TEXT PRIMARY KEY,
    wallet_id     TEXT NOT NULL REFERENCES wallets(id),
    txid          TEXT NOT NULL,
    vout          INTEGER NOT NULL,
    acquired_at   INTEGER NOT NULL,
    cost_sats     INTEGER NOT NULL DEFAULT 0,
    cost_fiat     INTEGER NOT NULL DEFAULT 0,
    fiat_currency TEXT NOT NULL DEFAULT '',
    method        TEXT NOT NULL CHECK (method IN ('fifo', 'avgcost')),
    disposed_at   INTEGER,
    proceeds_fiat INTEGER,
    gain_fiat     INTEGER
);

CREATE INDEX idx_cost_basis_wallet_id ON cost_basis_records(wallet_id);

CREATE TABLE price_snapshots (
    id         TEXT PRIMARY KEY,
    currency   TEXT NOT NULL,
    price_fiat INTEGER NOT NULL,
    source     TEXT NOT NULL DEFAULT 'manual',
    timestamp  INTEGER NOT NULL
);

CREATE INDEX idx_price_snapshots_currency_time ON price_snapshots(currency, timestamp);

CREATE TABLE counterparties (
    id         TEXT PRIMARY KEY,
    wallet_id  TEXT NOT NULL REFERENCES wallets(id),
    name       TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'unknown',
    note       TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);

-- BIP329-compatible labels
CREATE TABLE labels (
    id         TEXT PRIMARY KEY,
    wallet_id  TEXT NOT NULL REFERENCES wallets(id),
    type       TEXT NOT NULL CHECK (type IN ('tx', 'addr', 'xpub', 'input', 'output')),
    ref        TEXT NOT NULL,
    label      TEXT NOT NULL DEFAULT '',
    origin     TEXT NOT NULL DEFAULT '',
    spendable  INTEGER,
    created_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_labels_wallet_type_ref ON labels(wallet_id, type, ref);

CREATE TABLE tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE import_jobs (
    id               TEXT PRIMARY KEY,
    wallet_id        TEXT NOT NULL REFERENCES wallets(id),
    source           TEXT NOT NULL,
    filename         TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL CHECK (status IN ('pending', 'running', 'done', 'failed')),
    records_imported INTEGER NOT NULL DEFAULT 0,
    error_message    TEXT NOT NULL DEFAULT '',
    started_at       INTEGER,
    finished_at      INTEGER
);

CREATE INDEX idx_import_jobs_wallet_id ON import_jobs(wallet_id);
