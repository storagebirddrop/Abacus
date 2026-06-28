-- Prevent duplicate ledger entries for the same transaction + type + category.
-- Allows INSERT OR IGNORE to be idempotent on re-import.
CREATE UNIQUE INDEX IF NOT EXISTS idx_ledger_tx_type_category
    ON ledger_entries(transaction_id, type, category);
