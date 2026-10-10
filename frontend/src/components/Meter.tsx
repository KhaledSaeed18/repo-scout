import { cn } from '@/lib/utils'

/** Thin inline bar showing a value against the largest in its list. */
export default function Meter({ value, max, className }: { value: number; max: number; className?: string }) {
  const pct = max > 0 && value > 0 ? Math.max(2, (value / max) * 100) : 0
  return (
    <div aria-hidden className={cn('h-1 w-full overflow-hidden rounded-full bg-muted', className)}>
      <div className="h-full rounded-full bg-primary/70" style={{ width: `${pct}%` }} />
    </div>
  )
}
