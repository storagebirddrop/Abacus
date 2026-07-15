import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { getWallet, type Wallet } from '../api/wallets'
import { cn } from '../lib/utils'
import { TransactionsTab } from './wallet/TransactionsTab'
import { AccountingTab } from './wallet/AccountingTab'
import { AdvancedTab } from './wallet/AdvancedTab'

type Tab = 'transactions' | 'accounting' | 'advanced'

export default function WalletPage() {
  const { id } = useParams<{ id: string }>()
  const [wallet, setWallet] = useState<Wallet | null>(null)
  const [tab, setTab] = useState<Tab>('transactions')
  const [loadError, setLoadError] = useState('')

  useEffect(() => {
    if (!id) return
    const controller = new AbortController()
    setLoadError('')
    getWallet(id, controller.signal)
      .then(setWallet)
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === 'AbortError') return
        setLoadError(err instanceof Error ? err.message : 'Failed to load wallet')
      })
    return () => controller.abort()
  }, [id])

  if (!id) return null

  const tabs: { key: Tab; label: string }[] = [
    { key: 'transactions', label: 'Transactions' },
    { key: 'accounting', label: 'Accounting' },
    { key: 'advanced', label: 'Advanced' },
  ]

  return (
    <div className="page-surface p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">{wallet?.name ?? 'Wallet'}</h1>
        {wallet?.fingerprint && (
          <p className="text-sm text-muted-foreground font-mono mt-0.5">{wallet.fingerprint}</p>
        )}
        {loadError && (
          <p role="alert" className="text-sm text-destructive mt-1">{loadError}</p>
        )}
      </div>

      <div className="flex gap-1 border-b border-border mb-6" role="tablist" aria-label="Wallet sections">
        {tabs.map(({ key, label }) => (
          <button
            key={key}
            id={`tab-${key}`}
            role="tab"
            aria-selected={tab === key}
            aria-controls={`tabpanel-${key}`}
            onClick={() => setTab(key)}
            className={cn(
              'px-4 py-2 text-sm font-medium -mb-px border-b-2 transition-colors',
              tab === key
                ? 'border-border text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {tab === 'transactions' && (
        <div id="tabpanel-transactions" role="tabpanel" aria-labelledby="tab-transactions">
          <TransactionsTab walletID={id} />
        </div>
      )}
      {tab === 'accounting' && (
        <div id="tabpanel-accounting" role="tabpanel" aria-labelledby="tab-accounting">
          <AccountingTab walletID={id} />
        </div>
      )}
      {tab === 'advanced' && (
        <div id="tabpanel-advanced" role="tabpanel" aria-labelledby="tab-advanced">
          <AdvancedTab walletID={id} walletName={wallet?.name ?? ''} />
        </div>
      )}
    </div>
  )
}
