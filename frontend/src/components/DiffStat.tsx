import { formatNumber } from '@/lib/format'

const blocks = 5

/** Lines added and removed, with a five-block bar of their proportion. */
export default function DiffStat({ insertions, deletions }: { insertions: number; deletions: number }) {
  const total = insertions + deletions
  const added = total ? Math.round((insertions / total) * blocks) : 0
  return (
    <span className="inline-flex items-center gap-2 tabular-nums text-xs">
      <span className="text-success">+{formatNumber(insertions)}</span>
      <span className="text-destructive">−{formatNumber(deletions)}</span>
      <span aria-hidden className="flex gap-px">
        {Array.from({ length: blocks }, (_, i) => (
          <span
            key={i}
            className={
              'size-2 rounded-[1px] ' +
              (!total ? 'bg-muted' : i < added ? 'bg-success' : 'bg-destructive')
            }
          />
        ))}
      </span>
    </span>
  )
}
