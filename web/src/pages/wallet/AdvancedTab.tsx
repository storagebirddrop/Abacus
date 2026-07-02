import { useNavigate } from 'react-router-dom'
import { deleteWallet } from '../../api/wallets'
import { Button } from '../../components/ui/button'
import { useToast } from '../../components/Toast'
import { useConfirm } from '../../components/ConfirmDialog'
import { SyncPanel } from './SyncPanel'

export function AdvancedTab({ walletID, walletName }: { walletID: string; walletName: string }) {
  const navigate = useNavigate()
  const { toast } = useToast()
  const confirm = useConfirm()

  async function handleDelete() {
    const ok = await confirm({
      title: 'Delete wallet',
      description: `Delete "${walletName}"? This removes all transactions, ledger entries, and accounting data. This cannot be undone.`,
      confirmText: 'Delete',
      destructive: true,
    })
    if (!ok) return
    try {
      await deleteWallet(walletID)
      toast(`Deleted "${walletName}"`, 'success')
      navigate('/')
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : 'Delete failed', 'error')
    }
  }

  return (
    <div className="space-y-8">
      <section>
        <h2 className="text-base font-medium mb-4">Blockchain Sync</h2>
        <SyncPanel walletID={walletID} />
      </section>

      <section>
        <h2 className="text-base font-medium text-red-600 dark:text-red-400 mb-4">Danger Zone</h2>
        <div className="border border-red-200 dark:border-red-900 rounded-lg p-4">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-sm font-medium text-slate-800 dark:text-slate-100">Delete this wallet</p>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                Permanently removes all transactions, ledger entries, and accounting data.
              </p>
            </div>
            <Button
              variant="ghost"
              size="sm"
              aria-label={`Delete wallet ${walletName}`}
              onClick={handleDelete}
              className="shrink-0 text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950/30"
            >
              Delete wallet
            </Button>
          </div>
        </div>
      </section>
    </div>
  )
}
