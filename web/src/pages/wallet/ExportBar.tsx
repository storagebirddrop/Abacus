export function ExportBar({
  walletID,
  report,
}: {
  walletID: string
  report: 'transactions' | 'pnl' | 'balance-sheet'
}) {
  const base = `/api/v1/wallets/${walletID}/reports/${report}`
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-muted-foreground font-medium">Export:</span>
      {(['csv', 'xlsx', 'pdf'] as const).map((fmt) => (
        <a
          key={fmt}
          href={`${base}?format=${fmt}`}
          download
          className="text-xs px-2 py-1 rounded border border-border hover:bg-secondary text-muted-foreground uppercase font-mono"
        >
          {fmt}
        </a>
      ))}
    </div>
  )
}
