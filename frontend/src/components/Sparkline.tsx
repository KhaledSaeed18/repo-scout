import { useId, useState, type KeyboardEvent, type PointerEvent } from 'react'
import { nearestIndex, sparkPoints } from '@/lib/trend'

const width = 240
const height = 48
const pad = 5

/**
 * A single-series line with a crosshair that snaps to the nearest point on
 * hover, or moves with the arrow keys when focused. label(i) is the readout
 * for point i.
 */
export default function Sparkline({
  values,
  label,
  name,
}: {
  values: number[]
  label: (i: number) => string
  name: string
}) {
  const [active, setActive] = useState<number | null>(null)
  const tipId = useId()
  const pts = sparkPoints(values, width, height, pad)
  if (pts.length < 2) return null
  const path = pts.map((p, i) => `${i ? 'L' : 'M'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
  const last = pts[pts.length - 1]

  const onMove = (e: PointerEvent<SVGSVGElement>) => {
    const box = e.currentTarget.getBoundingClientRect()
    setActive(nearestIndex(((e.clientX - box.left) / box.width) * width, pts.length, width, pad))
  }
  const onKey = (e: KeyboardEvent<SVGSVGElement>) => {
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return
    e.preventDefault()
    const from = active ?? pts.length - 1
    setActive(Math.min(pts.length - 1, Math.max(0, from + (e.key === 'ArrowRight' ? 1 : -1))))
  }

  return (
    <div className="relative">
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="block h-auto w-full cursor-crosshair touch-none overflow-visible outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        role="img"
        aria-label={`${name} over the last ${values.length} scans. Use the arrow keys to read each scan.`}
        aria-describedby={active !== null ? tipId : undefined}
        tabIndex={0}
        onPointerMove={onMove}
        onPointerLeave={() => setActive(null)}
        onKeyDown={onKey}
        onBlur={() => setActive(null)}
      >
        <path d={path} fill="none" stroke="var(--primary)" strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" vectorEffect="non-scaling-stroke" />
        {active === null ? (
          <circle cx={last.x} cy={last.y} r={3} fill="var(--primary)" stroke="var(--background)" strokeWidth={2} vectorEffect="non-scaling-stroke" />
        ) : (
          <>
            <line x1={pts[active].x} x2={pts[active].x} y1={0} y2={height} stroke="var(--muted-foreground)" strokeWidth={1} vectorEffect="non-scaling-stroke" />
            <circle cx={pts[active].x} cy={pts[active].y} r={4} fill="var(--primary)" stroke="var(--background)" strokeWidth={2} vectorEffect="non-scaling-stroke" />
          </>
        )}
      </svg>
      {active !== null && (
        <div
          id={tipId}
          role="status"
          className="pointer-events-none absolute bottom-full z-10 mb-1 -translate-x-1/2 rounded-md border bg-popover px-2 py-1 text-xs whitespace-nowrap text-popover-foreground tabular-nums shadow-sm"
          style={{ left: `${Math.min(85, Math.max(15, (pts[active].x / width) * 100))}%` }}
        >
          {label(active)}
        </div>
      )}
    </div>
  )
}
