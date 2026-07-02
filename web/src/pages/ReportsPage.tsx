import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listWallets, type Wallet } from '../api/wallets'
import {
  runAccounting,
  getAccountingSummary,
  type AccountingSummary,
  type AccountingMethod,
} from '../api/accounting'
import { Button } from '../components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select'
import { cn } from '../lib/utils'

// Maps country to accounting method + currency + tax jurisdiction slug
const COUNTRY_CONFIG: Record<
  string,
  { label: string; method: AccountingMethod; currency: string; jurisdiction: string | null }
> = {
  us: { label: 'United States', method: 'fifo', currency: 'USD', jurisdiction: 'us' },
  uk: { label: 'United Kingdom', method: 'section104', currency: 'GBP', jurisdiction: 'uk' },
  de: { label: 'Germany', method: 'fifo', currency: 'EUR', jurisdiction: 'de' },
  nl: { label: 'Netherlands', method: 'fifo', currency: 'EUR', jurisdiction: 'nl' },
  other: { label: 'Other (FIFO / EUR)', method: 'fifo', currency: 'EUR', jurisdiction: null },
}

function fmtCents(cents: number | null, currency: string) {
  if (cents === null || cents === 0) return '—'
  const symbols: Record<string, string> = { EUR: '€', USD: '$', GBP: '£' }
  const sym = symbols[currency] ?? currency + ' '
  return `${sym}${(cents / 100).toLocaleString('en-US', { minimumFractionDigits: 2 })}`
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-4">
      <p className="text-xs text-slate-500 dark:text-slate-400">{label}</p>
      <p className="text-xl font-semibold mt-1">{value}</p>
    </div>
  )
}

export default function ReportsPage() {
  const [wallets, setWallets] = useState<Wallet[]>([])
  const [walletsLoading, setWalletsLoading] = useState(true)
  const [walletID, setWalletID] = useState('')
  const [country, setCountry] = useState('us')
  const [year, setYear] = useState(new Date().getFullYear() - 1)
  const [summary, setSummary] = useState<AccountingSummary | null>(null)
  const [running, setRunning] = useState(false)
  const [runError, setRunError] = useState('')
  const [advanced, setAdvanced] = useState(false)
  const [methodOverride, setMethodOverride] = useState<AccountingMethod | ''>('')

  const cfg = COUNTRY_CONFIG[country]
  const method: AccountingMethod = (methodOverride || cfg.method)
  const currency = cfg.currency

  const currentYear = new Date().getFullYear()
  const years = Array.from({ length: 6 }, (_, i) => currentYear - i)

  useEffect(() => {
    listWallets()
      .then((data) => {
        setWallets(data ?? [])
        if (data?.length) setWalletID(data[0].id)
      })
      .finally(() => setWalletsLoading(false))
  }, [])

  // Load existing summary when wallet changes
  useEffect(() => {
    if (!walletID) return
    setSummary(null)
    getAccountingSummary(walletID).then(setSummary).catch(() => {})
  }, [walletID])

  async function handleRun(e: React.FormEvent) {
    e.preventDefault()
    if (!walletID) return
    setRunning(true)
    setRunError('')
    try {
      const sum = await runAccounting(walletID, method, currency)
      setSummary(sum)
    } catch (err: unknown) {
      setRunError(err instanceof Error ? err.message : 'Accounting run failed')
    } finally {
      setRunning(false)
    }
  }

  const base = walletID ? `/api/v1/wallets/${walletID}/reports` : null

  return (
    <div className="p-8 max-w-3xl">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">Reports</h1>
        <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
          Select a wallet and your tax jurisdiction, then generate and download all reports.
        </p>
      </div>

      {walletsLoading ? (
        <p className="text-slate-500 dark:text-slate-400">Loading wallets…</p>
      ) : wallets.length === 0 ? (
        <div className="text-center py-16 text-slate-400">
          <p className="text-lg">No wallets yet</p>
          <p className="text-sm mt-1">
            <Link to="/wallets" className="underline hover:text-slate-600">Add a wallet</Link> to get started.
          </p>
        </div>
      ) : (
        <form onSubmit={handleRun} className="space-y-6">
          {/* Step 1: Wallet + year */}
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-5 space-y-4">
            <h2 className="text-sm font-medium text-slate-700 dark:text-slate-200">1. Select wallet and year</h2>
            <div className="flex flex-wrap gap-4">
              <div>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-300 mb-1">Wallet</label>
                <Select value={walletID} onValueChange={setWalletID}>
                  <SelectTrigger className="w-56">
                    <SelectValue placeholder="Choose wallet" />
                  </SelectTrigger>
                  <SelectContent>
                    {wallets.map((w) => (
                      <SelectItem key={w.id} value={w.id}>{w.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-300 mb-1">Tax year</label>
                <Select value={String(year)} onValueChange={(v) => setYear(Number(v))}>
                  <SelectTrigger className="w-28">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {years.map((y) => (
                      <SelectItem key={y} value={String(y)}>{y}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          {/* Step 2: Country / jurisdiction */}
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-5 space-y-4">
            <h2 className="text-sm font-medium text-slate-700 dark:text-slate-200">2. Tax jurisdiction</h2>
            <div>
              <label className="block text-xs font-medium text-slate-600 dark:text-slate-300 mb-1">Country of tax residence</label>
              <Select value={country} onValueChange={(v) => { setCountry(v); setMethodOverride('') }}>
                <SelectTrigger className="w-56">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(COUNTRY_CONFIG).map(([k, v]) => (
                    <SelectItem key={k} value={k}>{v.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              Method: <span className="font-medium text-slate-700 dark:text-slate-200">{method.toUpperCase()}</span>
              {' · '}Currency: <span className="font-medium text-slate-700 dark:text-slate-200">{currency}</span>
            </p>

            {/* Advanced method override */}
            <button
              type="button"
              onClick={() => setAdvanced((v) => !v)}
              className="text-xs text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 underline"
            >
              {advanced ? 'Hide advanced options' : 'Override method (advanced)'}
            </button>
            {advanced && (
              <div>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-300 mb-1">Method override</label>
                <Select
                  value={methodOverride || cfg.method}
                  onValueChange={(v) => setMethodOverride(v as AccountingMethod)}
                >
                  <SelectTrigger className="w-40">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="fifo">FIFO</SelectItem>
                    <SelectItem value="avgcost">Average Cost</SelectItem>
                    <SelectItem value="lifo">LIFO</SelectItem>
                    <SelectItem value="hifo">HIFO</SelectItem>
                    <SelectItem value="specificid">Specific ID</SelectItem>
                    <SelectItem value="section104">Section 104 (UK)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>

          {/* Step 3: Price data note */}
          <div className="text-sm text-slate-500 dark:text-slate-400 flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>
              Accounting uses BTC/{currency} price snapshots for fiat values.{' '}
              <Link to="/prices" className="underline hover:text-slate-600 dark:hover:text-slate-300">
                Manage price snapshots →
              </Link>
            </span>
          </div>

          {/* Run button */}
          <div className="flex items-center gap-3">
            <Button type="submit" disabled={running || !walletID}>
              {running ? 'Running…' : 'Run Accounting'}
            </Button>
            {runError && <p className="text-sm text-red-500">{runError}</p>}
          </div>

          {/* Summary + downloads (shown after run or if prior data exists) */}
          {summary && (
            <div className="space-y-5 pt-2">
              <div className="grid grid-cols-3 gap-4">
                <SummaryCard label="Total Cost" value={fmtCents(summary.total_cost_fiat, currency)} />
                <SummaryCard label="Unrealised Gain" value={fmtCents(summary.unrealised_gain_fiat, currency)} />
                <SummaryCard label="Realised Gain" value={fmtCents(summary.realised_gain_fiat, currency)} />
              </div>

              {base && (
                <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-5">
                  <h2 className="text-sm font-medium text-slate-700 dark:text-slate-200 mb-4">Download reports</h2>
                  <div className="space-y-3">
                    {cfg.jurisdiction && (
                      <ReportRow
                        label="Tax Report"
                        description={`${cfg.label} — ${year}`}
                        links={(['csv', 'pdf'] as const).map((fmt) => ({
                          fmt,
                          href: `${base}/tax?jurisdiction=${cfg.jurisdiction}&year=${year}&format=${fmt}`,
                        }))}
                      />
                    )}
                    <ReportRow
                      label="Profit & Loss"
                      description="Disposed UTXOs with cost, proceeds, and gain"
                      links={(['csv', 'xlsx', 'pdf'] as const).map((fmt) => ({
                        fmt,
                        href: `${base}/pnl?format=${fmt}`,
                      }))}
                    />
                    <ReportRow
                      label="Balance Sheet"
                      description="Unspent UTXOs with cost basis and current holdings"
                      links={(['csv', 'xlsx', 'pdf'] as const).map((fmt) => ({
                        fmt,
                        href: `${base}/balance-sheet?format=${fmt}`,
                      }))}
                    />
                    <ReportRow
                      label="Transactions"
                      description="Full transaction history"
                      links={(['csv', 'xlsx', 'pdf'] as const).map((fmt) => ({
                        fmt,
                        href: `${base}/transactions?format=${fmt}`,
                      }))}
                    />
                  </div>
                </div>
              )}
            </div>
          )}
        </form>
      )}
    </div>
  )
}

function ReportRow({
  label,
  description,
  links,
}: {
  label: string
  description: string
  links: { fmt: string; href: string }[]
}) {
  return (
    <div className="flex items-center justify-between py-2 border-b border-slate-100 dark:border-slate-800 last:border-0">
      <div>
        <p className="text-sm font-medium text-slate-800 dark:text-slate-100">{label}</p>
        <p className="text-xs text-slate-500 dark:text-slate-400">{description}</p>
      </div>
      <div className="flex gap-2">
        {links.map(({ fmt, href }) => (
          <a
            key={fmt}
            href={href}
            download
            className={cn(
              'text-xs px-2 py-1 rounded border uppercase font-mono',
              'border-slate-200 dark:border-slate-700',
              'hover:bg-slate-50 dark:hover:bg-slate-800',
              'text-slate-600 dark:text-slate-300',
            )}
          >
            {fmt}
          </a>
        ))}
      </div>
    </div>
  )
}
