import { Link } from 'react-router-dom'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import Meter from '@/components/Meter'
import RequireRepo from '@/components/RequireRepo'
import Sparkline from '@/components/Sparkline'
import { PageHeader, Section } from '@/components/layout'
import { QueryView } from '@/components/states'
import { useMetrics, useTrends } from '@/lib/api'
import { formatCompact, formatDate, formatNumber, formatPercent, shortHash } from '@/lib/format'
import { colorOf, rankLanguages } from '@/lib/languages'
import { describeChange } from '@/lib/trend'
import type { FileEntry, Metrics, Repository, ScanSnapshot } from '@/lib/types'

function Figure({ value, label, detail }: { value: string; label: string; detail?: string }) {
  return (
    <div className="flex flex-col gap-0.5 px-5 py-3 first:pl-0">
      <span className="font-heading text-3xl leading-none font-semibold tabular-nums">{value}</span>
      <span className="text-sm text-muted-foreground">{label}</span>
      {detail && <span className="text-xs text-muted-foreground">{detail}</span>}
    </div>
  )
}

function Totals({ m }: { m: Metrics }) {
  const t = m.totals
  const perFunction = t.funcs ? t.complexity / t.funcs : 0
  return (
    <div className="mb-2 flex flex-wrap divide-x border-y">
      <Figure value={formatCompact(t.code)} label="lines of code" detail={`${formatCompact(t.loc)} including blanks`} />
      <Figure value={formatCompact(t.comments)} label="comment lines" detail={`${formatPercent(t.code ? t.comments / (t.code + t.comments) : 0)} of written lines`} />
      <Figure value={formatCompact(t.funcs)} label="functions" />
      <Figure value={formatCompact(t.complexity)} label="total complexity" detail={t.funcs ? `${perFunction.toFixed(1)} per function` : undefined} />
      <Figure value={formatNumber(m.maxDepth)} label="deepest folder level" />
    </div>
  )
}

function Languages({ m }: { m: Metrics }) {
  const ranked = rankLanguages(m.languages)
  const rows = Object.entries(m.languages)
    .filter(([name]) => name !== '')
    .sort((a, b) => b[1].code - a[1].code || b[1].files - a[1].files)
  const max = rows[0]?.[1].code ?? 0
  const total = rows.reduce((s, [, v]) => s + v.code, 0)
  const unrecognized = m.languages['']?.files ?? 0
  return (
    <Section
      title="By language"
      description={unrecognized ? `${formatNumber(unrecognized)} files in formats Repo Scout does not count are left out.` : undefined}
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Language</TableHead>
            <TableHead className="text-right">Files</TableHead>
            <TableHead className="text-right">Lines of code</TableHead>
            <TableHead className="w-1/3">Share</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map(([name, v]) => (
            <TableRow key={name}>
              <TableCell>
                <span className="inline-flex items-center gap-2 font-medium">
                  <span className="size-2.5 rounded-[2px]" style={{ background: colorOf(ranked, name) }} />
                  {name}
                </span>
              </TableCell>
              <TableCell className="text-right">{formatNumber(v.files)}</TableCell>
              <TableCell className="text-right">{formatNumber(v.code)}</TableCell>
              <TableCell>
                <div className="flex items-center gap-3">
                  <Meter value={v.code} max={max} />
                  <span className="w-12 shrink-0 text-right text-xs text-muted-foreground">
                    {formatPercent(total ? v.code / total : 0)}
                  </span>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Section>
  )
}

function Ranked({
  title,
  description,
  files,
  value,
  unit,
}: {
  title: string
  description: string
  files: FileEntry[]
  value: (f: FileEntry) => number
  unit: string
}) {
  const shown = files.filter((f) => value(f) > 0)
  const max = shown[0] ? value(shown[0]) : 0
  return (
    <Section title={title} description={description}>
      {shown.length === 0 ? (
        <p className="text-sm text-muted-foreground">No source files were analyzed.</p>
      ) : (
        <ol className="flex flex-col gap-3">
          {shown.map((f) => (
            <li key={f.path} className="flex flex-col gap-1">
              <div className="flex items-baseline justify-between gap-3 text-sm">
                <Link
                  to={`/files?file=${encodeURIComponent(f.path)}`}
                  className="truncate font-mono text-[0.8125rem] hover:text-primary hover:underline"
                  title={f.path}
                >
                  {f.path}
                </Link>
                <span className="shrink-0 tabular-nums text-muted-foreground">
                  {formatNumber(value(f))} {unit}
                </span>
              </div>
              <Meter value={value(f)} max={max} />
            </li>
          ))}
        </ol>
      )}
    </Section>
  )
}

const trendMeasures: { key: keyof ScanSnapshot; label: string }[] = [
  { key: 'totalCode', label: 'Lines of code' },
  { key: 'complexity', label: 'Total complexity' },
  { key: 'functions', label: 'Functions' },
  { key: 'fileCount', label: 'Files' },
  { key: 'dupGroupCount', label: 'Duplicate blocks' },
  { key: 'dependencyCount', label: 'Dependencies' },
]

function TrendTile({ snapshots, measure }: { snapshots: ScanSnapshot[]; measure: (typeof trendMeasures)[number] }) {
  const values = snapshots.map((s) => Number(s[measure.key]))
  const current = values[values.length - 1]
  return (
    <div className="flex min-w-0 flex-col gap-1 border-b py-3">
      <span className="text-sm text-muted-foreground">{measure.label}</span>
      <span className="font-heading text-2xl leading-none font-semibold tabular-nums" title={formatNumber(current)}>
        {formatCompact(current)}
      </span>
      <span className="text-xs text-muted-foreground tabular-nums">{describeChange(values)}</span>
      <div className="mt-2">
        <Sparkline
          name={measure.label}
          values={values}
          label={(i) => `${formatDate(snapshots[i].scannedAt)}: ${formatNumber(values[i])}`}
        />
      </div>
    </div>
  )
}

function ScanTable({ snapshots }: { snapshots: ScanSnapshot[] }) {
  return (
    <Table className="mt-6">
      <TableHeader>
        <TableRow>
          <TableHead>Scanned</TableHead>
          <TableHead>Head commit</TableHead>
          {trendMeasures.map((m) => (
            <TableHead key={m.key} className="text-right">
              {m.label}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {[...snapshots].reverse().map((s) => (
          <TableRow key={s.id}>
            <TableCell className="whitespace-nowrap">{formatDate(s.scannedAt)}</TableCell>
            <TableCell className="font-mono text-[0.8125rem]">{s.headCommit ? shortHash(s.headCommit) : 'None'}</TableCell>
            {trendMeasures.map((m) => (
              <TableCell key={m.key} className="text-right">
                {formatNumber(Number(s[m.key]))}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function Trends({ repoId }: { repoId: number }) {
  const q = useTrends(repoId)
  return (
    <Section title="Over time" description="Each successful scan is kept, so scanning again after changes shows how the code is moving.">
      <QueryView query={q} label="scan history">
        {({ snapshots }) =>
          snapshots.length < 2 ? (
            <p className="text-sm text-muted-foreground">
              Only one scan so far. Scan again after the code changes to see trends here.
            </p>
          ) : (
            <>
              <div className="grid grid-cols-1 gap-x-8 sm:grid-cols-2 lg:grid-cols-3">
                {trendMeasures.map((m) => (
                  <TrendTile key={m.key} snapshots={snapshots} measure={m} />
                ))}
              </div>
              <ScanTable snapshots={snapshots} />
            </>
          )
        }
      </QueryView>
    </Section>
  )
}

function MetricsPage({ repo }: { repo: Repository }) {
  const q = useMetrics(repo.id, 20)
  return (
    <>
      <PageHeader
        title="Metrics"
        description={`How big and how complex the code in ${repo.name} is. Complexity counts decision points such as branches and loops.`}
      />
      <QueryView query={q} label="metrics">
        {(m) => (
          <>
            <Totals m={m} />
            <Languages m={m} />
            <Trends repoId={repo.id} />
            <div className="mt-10 grid grid-cols-1 gap-10 lg:grid-cols-2 [&>section]:mt-0">
              <Ranked title="Largest files" description="By lines of code." files={m.largestFiles} value={(f) => f.linesCode} unit="lines" />
              <Ranked
                title="Most complex files"
                description="By total complexity across the file."
                files={m.mostComplexFiles}
                value={(f) => f.complexity}
                unit=""
              />
            </div>
            {m.deepestFile && (
              <Section title="Deepest path">
                <p className="font-mono text-sm break-all">{m.deepestFile}</p>
              </Section>
            )}
          </>
        )}
      </QueryView>
    </>
  )
}

export default function MetricsRoute() {
  return <RequireRepo>{(repo) => <MetricsPage repo={repo} />}</RequireRepo>
}
