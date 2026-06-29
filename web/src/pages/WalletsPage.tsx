import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { createWallet, deleteWallet, listWallets, type Wallet } from '../api/wallets'
import { Button } from '../components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../components/ui/dialog'

function CreateWalletDialog({ onCreated }: { onCreated: () => void }) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [descriptor, setDescriptor] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await createWallet({ name, descriptor })
      setOpen(false)
      setName('')
      setDescriptor('')
      onCreated()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create wallet')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>New Wallet</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Wallet</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Name</label>
            <input
              className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-900"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My Bitcoin Wallet"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Output Descriptor</label>
            <textarea
              className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-slate-900 resize-none"
              value={descriptor}
              onChange={(e) => setDescriptor(e.target.value)}
              placeholder="wpkh([fingerprint/path]xpub...)"
              rows={4}
              required
            />
          </div>
          {error && <p className="text-sm text-red-500">{error}</p>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? 'Creating…' : 'Create'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default function WalletsPage() {
  const [wallets, setWallets] = useState<Wallet[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

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

  async function handleDelete(id: string, name: string) {
    if (!confirm(`Delete wallet "${name}"? This cannot be undone.`)) return
    try {
      await deleteWallet(id)
      setWallets((prev) => prev.filter((w) => w.id !== id))
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Delete failed')
    }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Wallets</h1>
        <CreateWalletDialog onCreated={load} />
      </div>

      {loading && <p className="text-slate-500">Loading…</p>}
      {error && <p className="text-red-500">{error}</p>}

      {!loading && wallets.length === 0 && (
        <div className="text-center py-16 text-slate-400">
          <p className="text-lg">No wallets yet</p>
          <p className="text-sm mt-1">Create a wallet or import a file to get started.</p>
        </div>
      )}

      {wallets.length > 0 && (
        <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Name</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Type</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Network</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Created</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {wallets.map((w) => (
                <tr key={w.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3">
                    <Link to={`/wallets/${w.id}`} className="font-medium text-slate-900 hover:underline">
                      {w.name}
                    </Link>
                    {w.fingerprint && (
                      <span className="ml-2 text-xs text-slate-400 font-mono">{w.fingerprint}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-slate-500 capitalize">{w.type || '—'}</td>
                  <td className="px-4 py-3 text-slate-500 capitalize">{w.network || '—'}</td>
                  <td className="px-4 py-3 text-slate-500">
                    {w.created_at ? new Date(w.created_at).toLocaleDateString() : '—'}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDelete(w.id, w.name)}
                      className="text-red-500 hover:text-red-600 hover:bg-red-50"
                    >
                      Delete
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
