import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listWallets, type Wallet } from '../api/wallets'
import { AddWalletDialog } from '../components/AddWalletDialog'

export default function WalletsPage() {
  const [wallets, setWallets] = useState<Wallet[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<'name'>('name')
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc')

  function toggleSort(key: 'name') {
    if (key === sortKey) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  const visible = wallets
    .filter((w) => {
      const q = search.trim().toLowerCase()
      if (!q) return true
      return w.name.toLowerCase().includes(q) || w.fingerprint.toLowerCase().includes(q)
    })
    .sort((a, b) => {
      const cmp = a.name.localeCompare(b.name)
      return sortDir === 'asc' ? cmp : -cmp
    })

  async function load() {
    try {
      const data = await listWallets()
      setWallets(data ?? [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load wallets')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  return (
    <div className="page-surface p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Wallets</h1>
        <AddWalletDialog onCreated={load} />
      </div>

      {loading && <p className="text-muted-foreground">Loading…</p>}
      {error && <p className="text-destructive">{error}</p>}

      {!loading && wallets.length === 0 && (
        <div className="text-center py-16 text-muted-foreground">
          <p className="text-lg">No wallets yet</p>
          <p className="text-sm mt-1">Add a wallet by importing a Sparrow, Nunchuk, Coldcard, or other export file.</p>
        </div>
      )}

      {wallets.length > 0 && (
        <>
          <input
            type="search"
            aria-label="Search wallets"
            placeholder="Search by name or fingerprint…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full sm:w-72 mb-4 border border-border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring bg-card"
          />

          {visible.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <p>No wallets match "{search}".</p>
            </div>
          ) : (
          <div className="bg-card border border-border rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-secondary/60 border-b border-border">
                <tr>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">
                    <button className="hover:text-foreground" onClick={() => toggleSort('name')}>
                      Name{sortKey === 'name' ? (sortDir === 'asc' ? ' ↑' : ' ↓') : ''}
                    </button>
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {visible.map((w) => (
                <tr key={w.id} className="hover:bg-secondary">
                  <td className="px-4 py-3">
                    <Link to={`/wallets/${w.id}`} className="font-medium text-foreground hover:underline">
                      {w.name}
                    </Link>
                    {w.fingerprint && (
                      <span className="ml-2 text-xs text-muted-foreground font-mono">{w.fingerprint}</span>
                    )}
                  </td>
                </tr>
              ))}
              </tbody>
            </table>
          </div>
          )}
        </>
      )}
    </div>
  )
}
