import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { getWallet, type Wallet } from '../api/wallets'
import { cn } from '../lib/utils'
import { TransactionsTab } from './wallet/TransactionsTab'
import { AccountingTab } from './wallet/AccountingTab'
import { ImportTab } from './wallet/ImportTab'
import { SyncPanel } from './wallet/SyncPanel'

type Tab = 'transactions' | 'accounting' | 'import' | 'sync'

export default function WalletPage() {
  const { id } = useParams<{ id: string }>()
  const [wallet, setWallet] = useState<Wallet | null>(null)
  const [tab, setTab] = useState<Tab>('transactions')
  const [loadError, setLoadError] = useState('')

  useEffect(() => {
    if (!id) return
    setLoadError('')
    getWallet(id)
      .then(setWallet)
      .catch((err: unknown) =>
        setLoadError(err instanceof Error ? err.message : 'Failed to load wallet'),
      )
  }, [id])

  if (!id) return null

  const tabs: { key: Tab; label: string }[] = [
    { key: 'transactions', label: 'Transactions' },
    { key: 'accounting', label: 'Accounting' },
    { key: 'import', label: 'Import' },
    { key: 'sync', label: 'Sync' },
  ]

  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">{wallet?.name ?? 'Wallet'}</h1>
        {wallet?.fingerprint && (
          <p className="text-sm text-slate-500 dark:text-slate-400 font-mono mt-0.5">{wallet.fingerprint}</p>
        )}
        {loadError && (
          <p role="alert" className="text-sm text-red-500 mt-1">{loadError}</p>
        )}
      </div>

      <div className="flex gap-1 border-b border-slate-200 dark:border-slate-800 mb-6">
        {tabs.map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={cn(
              'px-4 py-2 text-sm font-medium -mb-px border-b-2 transition-colors',
              tab === key
                ? 'border-slate-900 text-slate-900 dark:text-slate-100'
                : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-700'
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {tab === 'transactions' && <TransactionsTab walletID={id} />}
      {tab === 'accounting' && <AccountingTab walletID={id} />}
      {tab === 'import' && <ImportTab walletID={id} />}
      {tab === 'sync' && <SyncPanel walletID={id} />}
    </div>
  )
}
