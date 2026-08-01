import { cn } from '../../lib/utils'

export function Spinner({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        'size-4 animate-spin rounded-full border-2 border-slate-600 border-t-sky-400',
        className,
      )}
    />
  )
}
