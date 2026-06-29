CREATE TABLE IF NOT EXISTS sync_jobs (
    id TEXT PRIMARY KEY,
    wallet_id TEXT NOT NULL REFERENCES wallets(id),
    backend TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    addresses_scanned INTEGER NOT NULL DEFAULT 0,
    tx_found INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at INTEGER NOT NULL,
    finished_at INTEGER
);

CREATE TABLE IF NOT EXISTS sync_state (
    wallet_id TEXT PRIMARY KEY REFERENCES wallets(id),
    last_synced_at INTEGER NOT NULL,
    receive_gap_start INTEGER NOT NULL DEFAULT 0,
    change_gap_start INTEGER NOT NULL DEFAULT 0,
    block_height INTEGER NOT NULL DEFAULT 0
);
