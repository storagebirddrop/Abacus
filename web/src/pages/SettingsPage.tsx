import { useEffect, useState } from 'react'
import { getSettings, updateSettings, type AppSettings } from '../api/settings'
import { Button } from '../components/ui/button'

export default function SettingsPage() {
  const [settings, setSettings] = useState<AppSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    load()
  }, [])

  async function load() {
    setLoading(true)
    try {
      const data = await getSettings()
      setSettings(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load settings')
    } finally {
      setLoading(false)
    }
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!settings) return
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      const updated = await updateSettings(settings)
      setSettings(updated)
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  function set<K extends keyof AppSettings>(key: K, value: AppSettings[K]) {
    setSettings(prev => prev ? { ...prev, [key]: value } : prev)
  }

  if (loading) return <div className="p-8 text-slate-500">Loading…</div>
  if (!settings) return <div className="p-8 text-red-500">{error || 'Failed to load settings'}</div>

  return (
    <div className="p-8 max-w-xl">
      <h1 className="text-2xl font-bold text-slate-900 mb-1">Settings</h1>
      <p className="text-sm text-slate-500 mb-8">Configure blockchain sync and other preferences.</p>

      <form onSubmit={handleSave} className="space-y-8">
        {/* Blockchain Sync */}
        <section>
          <h2 className="text-base font-semibold text-slate-800 mb-4">Blockchain Sync</h2>

          <label className="flex items-center gap-3 mb-6 cursor-pointer">
            <input
              type="checkbox"
              className="w-4 h-4 accent-slate-700"
              checked={settings.sync_enabled}
              onChange={e => set('sync_enabled', e.target.checked)}
            />
            <div>
              <span className="text-sm font-medium text-slate-700">Enable blockchain sync</span>
              <p className="text-xs text-slate-400">
                When enabled, Abacus can fetch live transaction data from the Bitcoin network.
              </p>
            </div>
          </label>

          {settings.sync_enabled && (
            <div className="space-y-6 pl-7">
              {/* Privacy warning */}
              <div className="rounded border border-amber-300 bg-amber-50 px-3 py-2">
                <p className="text-xs text-amber-800">
                  <span className="font-semibold">Privacy notice:</span> syncing queries the
                  server below for your wallet's addresses, revealing them — and that they belong
                  to one wallet — to that third party. For maximum privacy, point this at an
                  Esplora or Electrum instance you host yourself. Abacus never sends keys or
                  signing material.
                </p>
              </div>

              {/* Backend selector */}
              <div>
                <p className="text-sm font-medium text-slate-700 mb-2">Backend</p>
                <div className="space-y-2">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="backend"
                      value="esplora"
                      className="accent-slate-700"
                      checked={settings.blockchain_backend === 'esplora'}
                      onChange={() => set('blockchain_backend', 'esplora')}
                    />
                    <span className="text-sm text-slate-700">Esplora API <span className="text-slate-400">(public or self-hosted REST)</span></span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="backend"
                      value="electrum"
                      className="accent-slate-700"
                      checked={settings.blockchain_backend === 'electrum'}
                      onChange={() => set('blockchain_backend', 'electrum')}
                    />
                    <span className="text-sm text-slate-700">Electrum server <span className="text-slate-400">(public or self-hosted)</span></span>
                  </label>
                </div>
              </div>

              {/* Esplora fields */}
              {settings.blockchain_backend === 'esplora' && (
                <div className="space-y-4">
                  <div>
                    <label className="block text-xs font-medium text-slate-600 mb-1">API URL</label>
                    <input
                      type="url"
                      className="w-full border border-slate-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-400"
                      value={settings.esplora_url}
                      onChange={e => set('esplora_url', e.target.value)}
                      placeholder="https://mempool.space/api"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-600 mb-1">
                      Rate limit <span className="font-normal text-slate-400">(ms between requests)</span>
                    </label>
                    <input
                      type="number"
                      min={0}
                      className="w-32 border border-slate-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-400"
                      value={settings.esplora_rate_ms}
                      onChange={e => set('esplora_rate_ms', Number(e.target.value))}
                    />
                  </div>
                </div>
              )}

              {/* Electrum fields */}
              {settings.blockchain_backend === 'electrum' && (
                <div className="space-y-4">
                  <div>
                    <label className="block text-xs font-medium text-slate-600 mb-1">Host</label>
                    <input
                      type="text"
                      className="w-full border border-slate-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-400"
                      value={settings.electrum_host}
                      onChange={e => set('electrum_host', e.target.value)}
                      placeholder="electrum.blockstream.info"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-600 mb-1">Port</label>
                    <input
                      type="number"
                      min={1}
                      max={65535}
                      className="w-32 border border-slate-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-400"
                      value={settings.electrum_port}
                      onChange={e => set('electrum_port', Number(e.target.value))}
                    />
                  </div>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      className="w-4 h-4 accent-slate-700"
                      checked={settings.electrum_tls}
                      onChange={e => set('electrum_tls', e.target.checked)}
                    />
                    <span className="text-sm text-slate-700">Use TLS</span>
                  </label>
                </div>
              )}
            </div>
          )}
        </section>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <div className="flex items-center gap-4">
          <Button type="submit" disabled={saving}>
            {saving ? 'Saving…' : 'Save settings'}
          </Button>
          {saved && <span className="text-sm text-green-600">Saved</span>}
        </div>
      </form>
    </div>
  )
}
