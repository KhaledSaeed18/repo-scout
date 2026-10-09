import type { UseQueryResult } from '@tanstack/react-query'
import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { Button, Spinner } from '@/components/ui'
import { cn } from '@/lib/utils'

export function Loading({ label, className }: { label: string; className?: string }) {
  return (
    <div role="status" className={cn('flex items-center gap-2 py-10 text-sm text-muted-foreground', className)}>
      <Spinner />
      {label}
    </div>
  )
}

export function ErrorNotice({
  title,
  error,
  onRetry,
}: {
  title: string
  error: unknown
  onRetry?: () => void
}) {
  const message = error instanceof Error ? error.message : String(error)
  return (
    <div role="alert" className="flex flex-wrap items-start justify-between gap-3 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm">
      <div>
        <p className="font-medium text-destructive">{title}</p>
        <p className="text-muted-foreground">{message}</p>
      </div>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          Try again
        </Button>
      )}
    </div>
  )
}

/** Quiet placeholder for an empty result. The action says what to do next. */
export function Empty({
  icon: Icon,
  title,
  children,
  className,
}: {
  icon: LucideIcon
  title: string
  children?: ReactNode
  className?: string
}) {
  return (
    <div className={cn('flex flex-col items-start gap-2 rounded-md border border-dashed px-5 py-8', className)}>
      <Icon className="size-5 text-muted-foreground" aria-hidden />
      <p className="font-medium">{title}</p>
      {children && <div className="max-w-[60ch] text-sm text-muted-foreground">{children}</div>}
    </div>
  )
}

/** Renders a query's data, or a consistent loading and error state. */
export function QueryView<T>({
  query,
  label,
  children,
}: {
  query: UseQueryResult<T>
  label: string
  children: (data: T) => ReactNode
}) {
  if (query.isPending) return <Loading label={`Loading ${label}…`} />
  if (query.isError) {
    return <ErrorNotice title={`Couldn't load ${label}`} error={query.error} onRetry={() => void query.refetch()} />
  }
  return <>{children(query.data)}</>
}
