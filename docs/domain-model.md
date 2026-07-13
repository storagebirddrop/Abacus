# Abacus Domain Model

## Entity Overview

```
Wallet ──┬── Address (many)
         ├── Transaction ──┬── TransactionInput (many)
         │                 └── TransactionOutput (many)
         ├── UTXO (many)
         ├── ExchangeTrade (many)          ← exchange wallets only
         ├── LedgerEntry ──── JournalEntry (many, audit trail)
         │        │  └──────── linked to Transaction OR ExchangeTrade
         │        └───────── CostBasisRecord (one, after accounting run)
         ├── Label (many, BIP329)
         ├── Counterparty (many)
         ├── ImportJob (many)
         └── SyncJob (many)

PriceSnapshot (global, not wallet-scoped)
SyncState (one per wallet)
```

## Entities

### Wallet
The root entity. Represents a Bitcoin wallet defined by its output descriptor.

| Field | Type | Description |
|---|---|---|
| id | UUID | Primary key |
| name | string | Human name |
| descriptor | string | Output descriptor (singlesig or multisig) |
| fingerprint | string | Master key fingerprint |
| type | enum | `singlesig` \| `multisig` |
| network | enum | `mainnet` \| `testnet` \| `signet` |
| source | enum | `sparrow` \| `nunchuk` \| `coldcard` \| `specter` \| `electrum` \| `bsms` \| `manual` \| `exchange` |

### Transaction
An immutable record of a confirmed or unconfirmed Bitcoin transaction.

| Field | Type | Description |
|---|---|---|
| txid | string | Bitcoin transaction ID |
| block_height | int | 0 = unconfirmed |
| block_time | unix ts | Confirmation time |
| fee_sats | int | Miner fee in satoshis |

### UTXO
An unspent transaction output. Tracks spending state.

| Field | Type | Description |
|---|---|---|
| txid + vout | string + int | Unique identifier |
| sats | int | Value in satoshis |
| spent | bool | True if spent |
| spent_txid | string | Spending transaction |
| label | string | User label |

### ExchangeTrade
A buy, sell, deposit, withdrawal, or Lightning payment on a custodial exchange.
One trade = one row; fiat value is recorded at trade time (no price lookup needed).

| Field | Type | Description |
|---|---|---|
| id | UUID | Primary key |
| wallet_id | UUID | Reference to Wallet |
| trade_type | enum | `buy` \| `sell` \| `fee` \| `deposit` \| `withdrawal` \| `lightning_receive` \| `lightning_send` |
| exchange | string | `bitvavo` \| `bitonic` \| `kraken` \| `coinbase` \| `strike` |
| external_id | string | Exchange's own order/trade ID (dedup key, nullable) |
| traded_at | timestamp | When the trade occurred |
| sats | int | BTC side in satoshis |
| fiat_amount | int | Fiat side in cents (absolute value) |
| fiat_currency | string | ISO currency (EUR, USD, …) |
| fee_sats | int | Fee in satoshis (0 if N/A) |
| fee_fiat | int | Fee in fiat cents (0 if N/A) |
| note | string | Notes from export |

### LedgerEntry ⚠️ Immutable
The core accounting record. **Never updated after creation.**
Linked to either a `Transaction` (on-chain) or an `ExchangeTrade` (custodial), never both.

| Field | Type | Description |
|---|---|---|
| transaction_id | UUID? | Reference to Transaction (nullable) |
| exchange_trade_id | UUID? | Reference to ExchangeTrade (nullable) |
| type | enum | `debit` \| `credit` |
| sats | int | Amount in satoshis |
| fiat_amount | int | Amount in cents |
| fiat_currency | string | ISO currency (EUR, USD, …) |
| category | enum | Transaction category |
| note | string | User note |

Categories: `income`, `expense`, `transfer`, `exchange`, `mining`, `donation`, `salary`, `gift`, `coinjoin`, `lightning`, `correction`, `fee`, `unknown`

### JournalEntry
Records every change to a LedgerEntry's metadata. The audit trail.

| Field | Type | Description |
|---|---|---|
| ledger_entry_id | UUID | Reference to LedgerEntry |
| field_changed | string | Which field was changed |
| old_value | string | Previous value |
| new_value | string | New value |
| reason | string | User-provided reason |

### CostBasisRecord
Result of an accounting run. One record per UTXO acquisition/disposal.

| Field | Type | Description |
|---|---|---|
| method | enum | `fifo` \| `avgcost` \| `lifo` \| `hifo` \| `specificid` \| `section104` |
| cost_sats | int | Acquisition cost in sats |
| cost_fiat | int | Acquisition cost in cents |
| disposed_at | timestamp | Null if still held |
| proceeds_fiat | int | Disposal proceeds in cents |
| gain_fiat | int | Realized gain/loss in cents |

`gain_fiat` is only ever set on disposal — held (`disposed_at` null) records have no
stored gain. `AccountingSummary` (`GET /accounting/summary`) and `PortfolioSummary`
(`GET /portfolio/summary`) are computed on read, not persisted: they mark held
records to the latest known `PriceSnapshot` to derive an unrealised gain, and total
cost basis only across still-held records (a disposed record's cost belongs to the
realised side, not current holdings). `PortfolioHistoryPoint` (`GET /portfolio/history`)
is likewise computed on read — a daily cumulative sats balance built from `LedgerEntry`
rows, not a stored time series.

### PriceSnapshot
Historical BTC price at a point in time. Used for fiat calculations.

| Field | Type | Description |
|---|---|---|
| currency | string | ISO currency code |
| price_fiat | int | Price in cents per BTC |
| source | string | `manual`, `coingecko`, etc. |
| timestamp | unix ts | Price timestamp |

### Label (BIP329)
Portable wallet labels. Compatible with BIP329 standard.

| Field | Type | Description |
|---|---|---|
| type | enum | `tx` \| `addr` \| `xpub` \| `input` \| `output` |
| ref | string | The thing being labeled |
| label | string | Human label |
| origin | string | Derivation origin |
| spendable | bool? | Null means unset |

### ImportJob
Tracks the state of a file import operation.

| Field | Type | Description |
|---|---|---|
| source | string | `sparrow` \| `nunchuk` \| `coldcard` \| `specter` \| `electrum` \| `bsms` \| `bip329` \| `descriptor` \| `bitvavo` \| `bitonic` \| `kraken` \| `coinbase` \| `strike` |
| filename | string | Original filename |
| status | enum | `pending` \| `running` \| `done` \| `failed` |
| records_imported | int | Count of imported records |
| error_message | string | Set on failure |

### SyncJob
Tracks the state of a blockchain sync operation.

| Field | Type | Description |
|---|---|---|
| wallet_id | UUID | Reference to Wallet |
| backend | string | `esplora` \| `electrum` |
| status | enum | `pending` \| `running` \| `done` \| `failed` |
| addresses_scanned | int | Number of addresses queried |
| tx_found | int | Number of transactions discovered |
| error_message | string | Set on failure |
| started_at | unix ts | Job start time |
| finished_at | unix ts? | Job completion time |

### SyncState
Persists the last-known sync position for gap-limit resumption.

| Field | Type | Description |
|---|---|---|
| wallet_id | UUID | Primary key (one per wallet) |
| last_synced_at | unix ts | Timestamp of last successful sync |
| receive_gap_start | int | Next receiving address index to scan |
| change_gap_start | int | Next change address index to scan |
| block_height | int | Chain tip at last sync |
