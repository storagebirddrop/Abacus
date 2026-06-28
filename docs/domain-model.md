# Abacus Domain Model

## Entity Overview

```
Wallet ──┬── Address (many)
         ├── Transaction ──┬── TransactionInput (many)
         │                 └── TransactionOutput (many)
         ├── UTXO (many)
         ├── LedgerEntry ──── JournalEntry (many, audit trail)
         │        └───────── CostBasisRecord (one, after accounting run)
         ├── Label (many, BIP329)
         ├── Counterparty (many)
         └── ImportJob (many)

PriceSnapshot (global, not wallet-scoped)
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
| source | enum | `sparrow` \| `nunchuk` \| `bsms` \| `manual` |

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

### LedgerEntry ⚠️ Immutable
The core accounting record. **Never updated after creation.**

| Field | Type | Description |
|---|---|---|
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
| method | enum | `fifo` \| `avgcost` |
| cost_sats | int | Acquisition cost in sats |
| cost_fiat | int | Acquisition cost in cents |
| disposed_at | timestamp | Null if still held |
| proceeds_fiat | int | Disposal proceeds in cents |
| gain_fiat | int | Realized gain/loss in cents |

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
| source | string | `sparrow` \| `nunchuk` \| `bsms` \| `bip329` |
| filename | string | Original filename |
| status | enum | `pending` \| `running` \| `done` \| `failed` |
| records_imported | int | Count of imported records |
| error_message | string | Set on failure |
