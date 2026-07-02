import { SyncPanel } from './SyncPanel'

export function AdvancedTab({ walletID }: { walletID: string }) {
  return (
    <div className="space-y-8">
      <section>
        <h2 className="text-base font-medium mb-4">Blockchain Sync</h2>
        <SyncPanel walletID={walletID} />
      </section>
    </div>
  )
}
