import { useEffect, useState } from 'react'
import { createPrice, listPrices, type PriceSnapshot } from '../api/prices'
import { Button } from '../components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select'

function AddPriceDialog({ currency, onCreated }: { currency: string; onCreated: () => void }) {
  const [open, setOpen] = useState(false)
  const [date, setDate] = useState('')
  const [price, setPrice] = useState('')
  const [source, setSource] = useState('manual')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const ts = Math.floor(new Date(date).getTime() / 1000)
      const priceCents = Math.round(parseFloat(price) * 100)
      await createPrice({ currency, price_fiat: priceCents, source, timestamp: ts })
      setOpen(false)
      setDate('')
      setPrice('')
      onCreated()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally {
      setLoading(false)
    }
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
              className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-900"
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
              className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-900"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              placeholder="30000.00"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Source</label>
            <input
              className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-900"
              value={source}
              onChange={(e) => setSource(e.target.value)}
              placeholder="manual"
            />
          </div>
          {error && <p className="text-sm text-red-500">{error}</p>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit" disabled={loading}>{loading ? 'Saving…' : 'Save'}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default function PricesPage() {
  const [currency, setCurrency] = useState('EUR')
  const [prices, setPrices] = useState<PriceSnapshot[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  async function load() {
    setLoading(true)
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
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Price Snapshots</h1>
        <div className="flex items-center gap-3">
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

      {loading && <p className="text-slate-500">Loading…</p>}
      {error && <p className="text-red-500">{error}</p>}

      {!loading && prices.length === 0 && (
        <div className="text-center py-16 text-slate-400">
          <p className="text-lg">No price snapshots</p>
          <p className="text-sm mt-1">Add manual prices so accounting can compute fiat values.</p>
        </div>
      )}

      {prices.length > 0 && (
        <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Date</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600">Price ({currency}/BTC)</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Source</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {prices.map((p) => (
                <tr key={p.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3 text-slate-700">
                    {new Date(p.timestamp).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right font-mono">
                    {(p.price_fiat / 100).toLocaleString('en-US', { minimumFractionDigits: 2 })}
                  </td>
                  <td className="px-4 py-3 text-slate-500">{p.source}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
