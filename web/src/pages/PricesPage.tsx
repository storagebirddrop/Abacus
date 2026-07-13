import { useEffect, useState } from 'react'
import { createPrice, fetchPricesFromCoinGecko, listPrices, type PriceSnapshot } from '../api/prices'
import { listWallets, type Wallet } from '../api/wallets'
import { Button } from '../components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select'
import { useDialog } from '../hooks/useDialog'

function AddPriceDialog({ currency, onCreated }: { currency: string; onCreated: () => void }) {
  const [date, setDate] = useState('')
  const [price, setPrice] = useState('')
  const [source, setSource] = useState('manual')
  const { open, setOpen, error, loading, submit } = useDialog(onCreated)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    await submit(
      () => {
        const ts = Math.floor(new Date(date).getTime() / 1000)
        const priceCents = Math.round(parseFloat(price) * 100)
        return createPrice({ currency, price_fiat: priceCents, source, timestamp: ts })
      },
      () => {
        setDate('')
        setPrice('')
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>Add Price</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Price Snapshot</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Date</label>
            <input
              type="date"
              className="w-full border border-border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Price ({currency}/BTC)</label>
            <input
              type="number"
              step="0.01"
              min="0"
              className="w-full border border-border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              placeholder="30000.00"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Source</label>
            <input
              className="w-full border border-border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
              value={source}
              onChange={(e) => setSource(e.target.value)}
              placeholder="manual"
            />
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit" disabled={loading}>{loading ? 'Saving…' : 'Save'}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function CoinGeckoFetchPanel({ currency, onFetched }: { currency: string; onFetched: () => void }) {
  const [wallets, setWallets] = useState<Wallet[]>([])
  const [walletID, setWalletID] = useState('')
  const [fetching, setFetching] = useState(false)
  const [result, setResult] = useState<{ fetched: number; skipped: number } | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    listWallets().then((ws) => {
      setWallets(ws ?? [])
      if (ws?.length) setWalletID(ws[0].id)
    })
  }, [])

  async function handleFetch() {
    if (!walletID) return
    setFetching(true)
    setResult(null)
    setError('')
    try {
      const res = await fetchPricesFromCoinGecko(walletID, currency)
      setResult(res)
      if (res.fetched > 0) onFetched()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to fetch prices')
    } finally {
      setFetching(false)
    }
  }

  if (!wallets.length) return null

  return (
    <div className="flex items-center gap-3">
      <Select value={walletID} onValueChange={setWalletID}>
        <SelectTrigger className="w-44">
          <SelectValue placeholder="Select wallet" />
        </SelectTrigger>
        <SelectContent>
          {wallets.map((w) => (
            <SelectItem key={w.id} value={w.id}>{w.name}</SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button variant="outline" onClick={handleFetch} disabled={fetching || !walletID}>
        {fetching ? 'Fetching…' : 'Fetch from CoinGecko'}
      </Button>
      {result && (
        <span className="text-sm text-muted-foreground">
          {result.fetched > 0
            ? `Fetched ${result.fetched} price${result.fetched !== 1 ? 's' : ''}`
            : 'All dates already covered'}
          {result.skipped > 0 && `, ${result.skipped} skipped`}
        </span>
      )}
      {error && <span className="text-sm text-destructive">{error}</span>}
    </div>
  )
}

export default function PricesPage() {
  const [currency, setCurrency] = useState('EUR')
  const [prices, setPrices] = useState<PriceSnapshot[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [sortKey, setSortKey] = useState<'date' | 'price'>('date')
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')

  function toggleSort(key: 'date' | 'price') {
    if (key === sortKey) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir(key === 'date' ? 'desc' : 'asc')
    }
  }

  const sorted = [...prices].sort((a, b) => {
    const cmp =
      sortKey === 'date'
        ? (a.timestamp || '').localeCompare(b.timestamp || '')
        : a.price_fiat - b.price_fiat
    return sortDir === 'asc' ? cmp : -cmp
  })

  async function load() {
    setLoading(true)
    // Clear stale rows/errors so a failed refetch can't leave the previous
    // currency's prices showing under the spinner.
    setError('')
    setPrices([])
    try {
      const data = await listPrices(currency)
      setPrices(data ?? [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [currency])

  return (
    <div className="page-surface p-8">
      <div className="flex flex-wrap items-center justify-between gap-4 mb-6">
        <h1 className="text-2xl font-semibold">Price Snapshots</h1>
        <div className="flex flex-wrap items-center gap-3">
          <CoinGeckoFetchPanel currency={currency} onFetched={load} />
          <div className="w-px h-6 bg-border" />
          <Select value={currency} onValueChange={setCurrency}>
            <SelectTrigger className="w-24">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="EUR">EUR</SelectItem>
              <SelectItem value="USD">USD</SelectItem>
              <SelectItem value="GBP">GBP</SelectItem>
            </SelectContent>
          </Select>
          <AddPriceDialog currency={currency} onCreated={load} />
        </div>
      </div>

      {loading && <p className="text-muted-foreground">Loading…</p>}
      {error && <p className="text-destructive">{error}</p>}

      {!loading && prices.length === 0 && (
        <div className="text-center py-16 text-muted-foreground">
          <p className="text-lg">No price snapshots</p>
          <p className="text-sm mt-1">Add manual prices so accounting can compute fiat values.</p>
        </div>
      )}

      {prices.length > 0 && (
        <div className="bg-card border border-border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-secondary/60 border-b border-border">
              <tr>
                <th
                  className="text-left px-4 py-3 font-medium text-muted-foreground"
                  aria-sort={sortKey === 'date' ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}
                >
                  <button className="hover:text-foreground" onClick={() => toggleSort('date')}>
                    Date{sortKey === 'date' ? (sortDir === 'asc' ? ' ↑' : ' ↓') : ''}
                  </button>
                </th>
                <th
                  className="text-right px-4 py-3 font-medium text-muted-foreground"
                  aria-sort={sortKey === 'price' ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}
                >
                  <button className="hover:text-foreground" onClick={() => toggleSort('price')}>
                    Price ({currency}/BTC){sortKey === 'price' ? (sortDir === 'asc' ? ' ↑' : ' ↓') : ''}
                  </button>
                </th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Source</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {sorted.map((p) => (
                <tr key={p.id} className="hover:bg-secondary">
                  <td className="px-4 py-3 text-foreground">
                    {new Date(p.timestamp).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right font-mono">
                    {(p.price_fiat / 100).toLocaleString('en-US', { minimumFractionDigits: 2 })}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{p.source}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
