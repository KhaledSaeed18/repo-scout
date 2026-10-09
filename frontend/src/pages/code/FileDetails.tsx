import { FileQuestion } from 'lucide-react'
import type { ReactNode } from 'react'
import { Empty, QueryView } from '@/components/states'
import { useFiles } from '@/lib/api'
import { formatBytes, formatDate, formatNumber, formatRelative } from '@/lib/format'
import type { FileEntry } from '@/lib/types'

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex justify-between gap-4 border-b border-border/60 py-1.5 text-sm last:border-b-0">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="text-right tabular-nums">{children}</dd>
    </div>
  )
}

function Composition({ f }: { f: FileEntry }) {
  const parts = [
    { label: 'Code', value: f.linesCode, color: 'var(--chart-1)' },
    { label: 'Comments', value: f.linesComment, color: 'var(--chart-3)' },
    { label: 'Blank', value: f.linesBlank, color: 'var(--border)' },
  ]
  const total = f.linesTotal || 1
  return (
    <div className="flex flex-col gap-2">
      <div className="flex h-2 overflow-hidden rounded-full bg-muted" aria-hidden>
        {parts.map((p) => (
          <div key={p.label} style={{ width: `${(p.value / total) * 100}%`, background: p.color }} />
        ))}
      </div>
      <div className="flex flex-wrap gap-x-4 text-xs text-muted-foreground">
        {parts.map((p) => (
          <span key={p.label} className="flex items-center gap-1.5">
            <span className="size-2 rounded-[2px]" style={{ background: p.color }} />
            {p.label} {formatNumber(p.value)}
          </span>
        ))}
      </div>
    </div>
  )
}

function Details({ f }: { f: FileEntry }) {
  const analyzed = f.funcCount > 0 || f.complexity > 0
  return (
    <div className="flex flex-col gap-5">
      <div className="min-w-0">
        <h2 className="truncate text-xl font-semibold" title={f.name}>
          {f.name}
        </h2>
        <p className="font-mono text-xs break-all text-muted-foreground">{f.path}</p>
      </div>
      {f.linesTotal > 0 && <Composition f={f} />}
      <dl>
        <Row label="Language">{f.language || 'Not recognized'}</Row>
        <Row label="Lines">{formatNumber(f.linesTotal)}</Row>
        <Row label="Size">{formatBytes(f.size)}</Row>
        {analyzed && (
          <>
            <Row label="Complexity">{formatNumber(f.complexity)}</Row>
            <Row label="Functions">{formatNumber(f.funcCount)}</Row>
            <Row label="Average function length">{f.avgFuncLen.toFixed(1)} lines</Row>
            <Row label="Longest function">{formatNumber(f.maxFuncLen)} lines</Row>
            <Row label="Deepest nesting">{formatNumber(f.maxNesting)}</Row>
          </>
        )}
        <Row label="Imports">{formatNumber(f.imports)}</Row>
        <Row label="Exports">{formatNumber(f.exports)}</Row>
      </dl>
      <dl>
        <Row label="Commits touching it">{formatNumber(f.commits)}</Row>
        <Row label="Last changed by">{f.author || 'Unknown'}</Row>
        <Row label="First commit">{f.firstCommitAt ? formatDate(f.firstCommitAt) : 'Untracked'}</Row>
        <Row label="Last commit">{f.lastCommitAt ? formatRelative(f.lastCommitAt) : 'Untracked'}</Row>
      </dl>
    </div>
  )
}

/** Metrics and history of one file, looked up by path. */
export default function FileDetails({ repoId, path }: { repoId: number; path: string }) {
  const q = useFiles(repoId, { path, limit: 1 })
  return (
    <QueryView query={q} label="file details">
      {({ files }) =>
        files[0] ? (
          <Details f={files[0]} />
        ) : (
          <Empty icon={FileQuestion} title="File not found">
            It may have been removed since the last scan.
          </Empty>
        )
      }
    </QueryView>
  )
}
