export function ExportBar({
  walletID,
  report,
}: {
  walletID: string
  report: 'transactions' | 'pnl' | 'balance-sheet'
}) {
  const base = `/api/v1/wallets/${walletID}/reports/${report}`
  return (
    <div className="flex items-center gap-2 mb-4">
      <span className="text-xs text-slate-500 dark:text-slate-400 font-medium">Export:</span>
      {(['csv', 'xlsx', 'pdf'] as const).map((fmt) => (
        <a
          key={fmt}
          href={`${base}?format=${fmt}`}
          download
          className="text-xs px-2 py-1 rounded border border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-300 uppercase font-mono"
        >
          {fmt}
        </a>
      ))}
    </div>
  )
}
