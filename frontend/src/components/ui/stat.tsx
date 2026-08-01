import type { ReactNode } from 'react'
import { Card, CardContent } from './card'

export function Stat({
  label,
  value,
  sub,
}: {
  label: string
  value: ReactNode
  sub?: string
}) {
  return (
    <Card>
      <CardContent className="flex flex-col gap-1 pt-6">
        <span className="text-xs uppercase tracking-wide text-slate-500">{label}</span>
        <span className="text-2xl font-semibold text-slate-100">{value}</span>
        {sub && <span className="text-xs text-slate-400">{sub}</span>}
      </CardContent>
    </Card>
  )
}
