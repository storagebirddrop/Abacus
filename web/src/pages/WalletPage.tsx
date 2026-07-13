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
    { key: 'advanced', label: 'Advanced' },
  ]

  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">{wallet?.name ?? 'Wallet'}</h1>
        {wallet?.fingerprint && (
          <p className="text-sm text-muted-foreground font-mono mt-0.5">{wallet.fingerprint}</p>
        )}
        {loadError && (
          <p role="alert" className="text-sm text-destructive mt-1">{loadError}</p>
        )}
      </div>

      <div className="flex gap-1 border-b border-border mb-6">
        {tabs.map(({ key, label }) => (
          <button
            key={key}
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

      {tab === 'transactions' && <TransactionsTab walletID={id} />}
      {tab === 'accounting' && <AccountingTab walletID={id} />}
      {tab === 'advanced' && <AdvancedTab walletID={id} walletName={wallet?.name ?? ''} />}
    </div>
  )
}
