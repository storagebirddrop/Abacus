import { useEffect, useMemo, useRef, useState } from 'react'
import { getPortfolioHistory, type PortfolioHistoryPoint } from '../api/portfolio'

const RANGES = [
  { label: '30D', days: 30 },
  { label: '90D', days: 90 },
  { label: '180D', days: 180 },
  { label: '1Y', days: 365 },
] as const

const CURRENCY_SYMBOLS: Record<string, string> = { EUR: '€', USD: '$', GBP: '£' }

function fmtFiat(cents: number, currency: string): string {
  const sym = CURRENCY_SYMBOLS[currency] ?? currency + ' '
  const value = cents / 100
  if (Math.abs(value) >= 1000) {
    return `${sym}${(value / 1000).toFixed(1)}K`
  }
  return `${sym}${value.toFixed(0)}`
}

function fmtBTC(sats: number): string {
  return (sats / 1e8).toFixed(4) + ' BTC'
}

function fmtDateShort(iso: string): string {
  const d = new Date(iso + 'T00:00:00Z')
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' })
}

// Round a max value up to a "clean" tick ceiling (1/2/5 * 10^n).
function niceCeiling(max: number): number {
  if (max <= 0) return 1
  const magnitude = 10 ** Math.floor(Math.log10(max))
  const steps = [1, 2, 5, 10]
  for (const s of steps) {
    if (max <= s * magnitude) return s * magnitude
  }
  return 10 * magnitude
}

const WIDTH = 900
const HEIGHT = 260
const PAD = { top: 16, right: 16, bottom: 28, left: 56 }

export function PortfolioChart({ currency }: { currency: string }) {
  const [days, setDays] = useState<number>(90)
  const [metric, setMetric] = useState<'value' | 'sats'>('value')
  const [points, setPoints] = useState<PortfolioHistoryPoint[] | null>(null)
  const [error, setError] = useState('')
  const [hoverIdx, setHoverIdx] = useState<number | null>(null)
  const svgRef = useRef<SVGSVGElement>(null)

  useEffect(() => {
    setError('')
    getPortfolioHistory(currency, days)
      .then(setPoints)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Failed to load history'))
  }, [currency, days])

  const hasPrices = points?.some((p) => p.fiat_value > 0) ?? false
  const effectiveMetric = metric === 'value' && !hasPrices ? 'sats' : metric

  const chart = useMemo(() => {
    if (!points || points.length < 2) return null
    const values = points.map((p) => (effectiveMetric === 'value' ? p.fiat_value : p.total_sats))
    const maxRaw = Math.max(...values, 0)
    const max = niceCeiling(maxRaw || 1)
    const innerW = WIDTH - PAD.left - PAD.right
    const innerH = HEIGHT - PAD.top - PAD.bottom
    const x = (i: number) => PAD.left + (i / (points.length - 1)) * innerW
    const y = (v: number) => PAD.top + innerH - (v / max) * innerH
    const linePath = points.map((_, i) => `${i === 0 ? 'M' : 'L'} ${x(i).toFixed(2)} ${y(values[i]).toFixed(2)}`).join(' ')
    const areaPath = `${linePath} L ${x(points.length - 1).toFixed(2)} ${y(0).toFixed(2)} L ${x(0).toFixed(2)} ${y(0).toFixed(2)} Z`
    return { values, max, x, y, linePath, areaPath, innerH }
  }, [points, effectiveMetric])

  function handlePointerMove(e: React.PointerEvent<SVGSVGElement>) {
    if (!points || !chart || !svgRef.current) return
    const rect = svgRef.current.getBoundingClientRect()
    const px = ((e.clientX - rect.left) / rect.width) * WIDTH
    const innerW = WIDTH - PAD.left - PAD.right
    const ratio = Math.min(1, Math.max(0, (px - PAD.left) / innerW))
    const idx = Math.round(ratio * (points.length - 1))
    setHoverIdx(idx)
  }

  return (
    <div className="bg-card border border-border rounded-lg p-5">
      <div className="flex items-center justify-between mb-4 flex-wrap gap-3">
        <div>
          <h2 className="text-sm font-semibold text-foreground">Portfolio history</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            {effectiveMetric === 'value' ? `Value (${currency})` : 'BTC holdings'} across all wallets
          </p>
        </div>
        <div className="flex items-center gap-4">
          {hasPrices && (
            <div className="flex rounded-md border border-border overflow-hidden text-xs">
              {(['value', 'sats'] as const).map((m) => (
                <button
                  key={m}
                  onClick={() => setMetric(m)}
                  className={
                    effectiveMetric === m
                      ? 'px-2.5 py-1 bg-primary/15 text-primary font-medium'
                      : 'px-2.5 py-1 text-muted-foreground hover:bg-secondary'
                  }
                >
                  {m === 'value' ? currency : 'BTC'}
                </button>
              ))}
            </div>
          )}
          <div className="flex rounded-md border border-border overflow-hidden text-xs">
            {RANGES.map((r) => (
              <button
                key={r.days}
                onClick={() => setDays(r.days)}
                className={
                  days === r.days
                    ? 'px-2.5 py-1 bg-primary/15 text-primary font-medium'
                    : 'px-2.5 py-1 text-muted-foreground hover:bg-secondary'
                }
              >
                {r.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}

      {!error && points && points.length < 2 && (
        <p className="text-sm text-muted-foreground py-12 text-center">
          Not enough history yet — import wallet data to see a chart here.
        </p>
      )}

      {!error && chart && points && (
        <div className="relative">
          <svg
            ref={svgRef}
            viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
            className="w-full h-auto touch-none"
            onPointerMove={handlePointerMove}
            onPointerLeave={() => setHoverIdx(null)}
            role="img"
            aria-label={`Portfolio ${effectiveMetric === 'value' ? 'value' : 'BTC holdings'} over the last ${days} days`}
          >
            {/* Gridlines — recessive hairlines at 0%, 50%, 100% */}
            {[0, 0.5, 1].map((f) => (
              <line
                key={f}
                x1={PAD.left}
                x2={WIDTH - PAD.right}
                y1={PAD.top + chart.innerH * (1 - f)}
                y2={PAD.top + chart.innerH * (1 - f)}
                stroke="hsl(var(--border))"
                strokeWidth={1}
              />
            ))}

            {/* Y-axis labels */}
            {[0, 0.5, 1].map((f) => (
              <text
                key={f}
                x={PAD.left - 8}
                y={PAD.top + chart.innerH * (1 - f) + 4}
                textAnchor="end"
                className="fill-muted-foreground"
                fontSize={11}
              >
                {effectiveMetric === 'value' ? fmtFiat(chart.max * f, currency) : fmtBTC(chart.max * f)}
              </text>
            ))}

            {/* Area fill */}
            <path d={chart.areaPath} fill="hsl(var(--primary) / 0.1)" />
            {/* Line */}
            <path d={chart.linePath} fill="none" stroke="hsl(var(--primary))" strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />

            {/* X-axis labels: first, middle, last */}
            {[0, Math.floor((points.length - 1) / 2), points.length - 1].map((i) => (
              <text
                key={i}
                x={chart.x(i)}
                y={HEIGHT - 8}
                textAnchor={i === 0 ? 'start' : i === points.length - 1 ? 'end' : 'middle'}
                className="fill-muted-foreground"
                fontSize={11}
              >
                {fmtDateShort(points[i].date)}
              </text>
            ))}

            {/* Crosshair + hover point */}
            {hoverIdx !== null && (
              <>
                <line
                  x1={chart.x(hoverIdx)}
                  x2={chart.x(hoverIdx)}
                  y1={PAD.top}
                  y2={PAD.top + chart.innerH}
                  stroke="hsl(var(--muted-foreground))"
                  strokeWidth={1}
                  strokeDasharray="3,3"
                />
                <circle
                  cx={chart.x(hoverIdx)}
                  cy={chart.y(chart.values[hoverIdx])}
                  r={4}
                  fill="hsl(var(--primary))"
                  stroke="hsl(var(--card))"
                  strokeWidth={2}
                />
              </>
            )}
          </svg>

          {hoverIdx !== null && points[hoverIdx] && (
            <div
              className="absolute top-0 bg-popover text-popover-foreground border border-border rounded-md px-2.5 py-1.5 text-xs shadow-lg pointer-events-none"
              style={{
                left: `${(chart.x(hoverIdx) / WIDTH) * 100}%`,
                transform: hoverIdx > points.length / 2 ? 'translateX(-105%)' : 'translateX(5%)',
              }}
            >
              <p className="text-muted-foreground">{fmtDateShort(points[hoverIdx].date)}</p>
              <p className="font-semibold tabular-nums">
                {effectiveMetric === 'value'
                  ? fmtFiat(points[hoverIdx].fiat_value, currency)
                  : fmtBTC(points[hoverIdx].total_sats)}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
