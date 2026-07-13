import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowUpDown, FileDown, Upload, Wifi } from 'lucide-react'
import { getPortfolioSummary, type PortfolioSummary, type WalletSummary } from '../api/portfolio'
import { AddWalletDialog } from '../components/AddWalletDialog'
import { PortfolioChart } from '../components/PortfolioChart'
import { cn } from '../lib/utils'

function fmtBTC(sats: number): string {
  if (sats === 0) return '0 BTC'
  const btc = sats / 1e8
  return btc.toFixed(8).replace(/\.?0+$/, '') + ' BTC'
}

function fmtCents(cents: number, currency: string): string {
  if (cents === 0) return '—'
  const symbols: Record<string, string> = { EUR: '€', USD: '$', GBP: '£' }
  const sym = symbols[currency] ?? currency + ' '
  return `${sym}${(cents / 100).toLocaleString('en-US', { minimumFractionDigits: 2 })}`
}

function GainLabel({ cents, currency }: { cents: number; currency: string }) {
  if (cents === 0) return <span className="text-muted-foreground">—</span>
  const isPos = cents > 0
  return (
    <span className={isPos ? 'text-success' : 'text-destructive'}>
      {isPos ? '+' : ''}{fmtCents(cents, currency)}
    </span>
  )
}

function SummaryCard({ label, value, sub }: { label: string; value: string; sub?: React.ReactNode }) {
  return (
    <div className="bg-card border border-border rounded-lg p-5">
      <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">{label}</p>
      <p className="text-2xl font-semibold mt-1 text-foreground">{value}</p>
      {sub && <p className="text-sm mt-0.5">{sub}</p>}
    </div>
  )
}

function QuickAction({
  to, icon: Icon, label, description,
}: {
  to: string
  icon: React.ComponentType<{ size?: number; className?: string }>
  label: string
  description: string
}) {
  return (
    <Link
      to={to}
      className="flex items-start gap-3 bg-card border border-border rounded-lg p-4 hover:border-primary/50 transition-colors group"
    >
      <div className="shrink-0 h-9 w-9 rounded-md bg-primary/10 text-primary flex items-center justify-center group-hover:bg-primary/15">
        <Icon size={18} />
      </div>
      <div>
        <p className="text-sm font-medium text-foreground">{label}</p>
        <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
      </div>
    </Link>
  )
}

function WalletRow({ w }: { w: WalletSummary }) {
  const hasAccounting = Boolean(w.method)
  const currency = w.fiat_currency ?? 'EUR'

  return (
    <tr className="hover:bg-secondary">
      <td className="px-4 py-3">
        <Link
          to={`/wallets/${w.wallet_id}`}
          className="font-medium text-foreground hover:underline"
        >
          {w.wallet_name}
        </Link>
      </td>
      <td className="px-4 py-3 text-right font-medium tabular-nums text-foreground">
        {fmtBTC(w.total_sats)}
      </td>
      <td className="px-4 py-3 text-right tabular-nums">
        {hasAccounting ? (
          <GainLabel cents={w.unrealised_gain_fiat} currency={currency} />
        ) : (
          <span className="text-xs text-muted-foreground">no accounting</span>
        )}
      </td>
      <td className="px-4 py-3 text-right tabular-nums">
        {hasAccounting ? (
          <GainLabel cents={w.realised_gain_fiat} currency={currency} />
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        )}
      </td>
    </tr>
  )
}

export default function PortfolioPage() {
  const [summary, setSummary] = useState<PortfolioSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  function load() {
    setLoading(true)
    setError('')
    getPortfolioSummary()
      .then(setSummary)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Failed to load portfolio'))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [])

  const hasWallets = summary !== null && summary.wallet_count > 0
  const hasAccounting = hasWallets && summary.wallets.some((w) => w.method)
  const currency = hasAccounting
    ? (summary.wallets.find((w) => w.fiat_currency)?.fiat_currency ?? 'EUR')
    : 'EUR'

  return (
    <div className="page-surface p-8 max-w-6xl">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Portfolio</h1>
        <AddWalletDialog onCreated={load} />
      </div>

      {loading && <p className="text-muted-foreground">Loading…</p>}
      {error && <p className="text-destructive">{error}</p>}

      {!loading && summary !== null && (
        <>
          {/* Summary cards */}
          {hasWallets ? (
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
              <SummaryCard
                label="Total Holdings"
                value={fmtBTC(summary.total_sats)}
              />
              <SummaryCard
                label="Total Cost"
                value={hasAccounting ? fmtCents(summary.total_cost_fiat, currency) : '—'}
              />
              <SummaryCard
                label="Unrealised Gain"
                value={hasAccounting ? '' : '—'}
                sub={hasAccounting ? (
                  <GainLabel cents={summary.unrealised_gain_fiat} currency={currency} />
                ) : undefined}
              />
              <SummaryCard
                label="Realised Gain"
                value={hasAccounting ? '' : '—'}
                sub={hasAccounting ? (
                  <GainLabel cents={summary.realised_gain_fiat} currency={currency} />
                ) : undefined}
              />
            </div>
          ) : (
            <div className="text-center py-16 text-muted-foreground">
              <p className="text-lg">No wallets yet</p>
              <p className="text-sm mt-1">
                Add a wallet by importing a Sparrow, Nunchuk, Coldcard, or other export file.
              </p>
            </div>
          )}

          {/* Wallet table */}
          {hasWallets && (
            <div className="bg-card border border-border rounded-lg overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-secondary/60 border-b border-border">
                  <tr>
                    <th className="text-left px-4 py-3 font-medium text-muted-foreground">Wallet</th>
                    <th className="text-right px-4 py-3 font-medium text-muted-foreground">Holdings</th>
                    <th className={cn(
                      'text-right px-4 py-3 font-medium text-muted-foreground',
                      !hasAccounting && 'opacity-40'
                    )}>
                      Unrealised Gain
                    </th>
                    <th className={cn(
                      'text-right px-4 py-3 font-medium text-muted-foreground',
                      !hasAccounting && 'opacity-40'
                    )}>
                      Realised Gain
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {summary.wallets.map((w) => (
                    <WalletRow key={w.wallet_id} w={w} />
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Accounting note */}
          {hasWallets && !hasAccounting && (
            <p className="text-xs text-muted-foreground mt-3">
              Run accounting on a wallet to see fiat gain/loss figures.
            </p>
          )}

          {/* History chart */}
          {hasWallets && (
            <div className="mt-6">
              <PortfolioChart currency={currency} />
            </div>
          )}

          {/* Quick actions */}
          {hasWallets && (
            <div className="mt-10">
              <h2 className="text-sm font-semibold text-foreground mb-3">Quick actions</h2>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                <QuickAction
                  to={`/wallets/${summary.wallets[0].wallet_id}`}
                  icon={Upload}
                  label="Import data"
                  description="Add transaction history to a wallet from a file export."
                />
                <QuickAction
                  to={`/wallets/${summary.wallets[0].wallet_id}`}
                  icon={ArrowUpDown}
                  label="Run accounting"
                  description="Compute cost basis and gain/loss for a wallet."
                />
                <QuickAction
                  to="/reports"
                  icon={FileDown}
                  label="Generate reports"
                  description="Tax, P&L, and balance sheet reports as CSV, PDF, or Excel."
                />
                <QuickAction
                  to="/settings"
                  icon={Wifi}
                  label="Enable blockchain sync"
                  description="Pull live transaction history from Esplora or Electrum."
                />
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
