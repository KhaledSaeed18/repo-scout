import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { Card, Empty, Spinner, Stat, Table } from '../components/ui'
import RepoSelector from '../components/RepoSelector'
import { useMetrics, useRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'

const colors = [
  '#38bdf8',
  '#f59e0b',
  '#34d399',
  '#f87171',
  '#a78bfa',
  '#22d3ee',
  '#fb923c',
  '#4ade80',
  '#e879f9',
]

function MetricsPage() {
  const { repoId } = useRepoContext()
  const repo = useRepo(repoId).data
  const { data, isLoading } = useMetrics(repoId)
  if (repoId === 0) return <Empty message="Scan a repository first" />
  if (isLoading) return <Spinner />
  if (!data) return <Empty message="No metrics yet" />

  const langData = Object.entries(data.languages)
    .filter(([, v]) => v.files > 0)
    .sort((a, b) => b[1].code - a[1].code)
    .slice(0, 8)
    .map(([name, v]) => ({ name, code: v.code }))

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Metrics</h1>
        <RepoSelector />
      </div>
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <Stat label="Files" value={data.totals.files} />
        <Stat label="Code lines" value={data.totals.code.toLocaleString()} />
        <Stat label="Complexity" value={data.totals.complexity} />
        <Stat label="Functions" value={data.totals.funcs} />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <h3 className="mb-3 text-sm font-medium text-slate-300">Lines of code by language</h3>
          {langData.length ? (
            <ResponsiveContainer width="100%" height={240}>
              <BarChart data={langData}>
                <XAxis dataKey="name" stroke="#64748b" fontSize={11} />
                <YAxis stroke="#64748b" fontSize={12} />
                <Tooltip
                  contentStyle={{ background: '#0f172a', border: '1px solid #334155' }}
                />
                <Bar dataKey="code" radius={[4, 4, 0, 0]}>
                  {langData.map((_, i) => (
                    <Cell key={i} fill={colors[i % colors.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <Empty message="No source files" />
          )}
        </Card>
        <Card className="flex flex-col gap-3">
          <h3 className="text-sm font-medium text-slate-300">Aggregates</h3>
          <Table
            headers={['Metric', 'Value']}
            rows={[
              ['Comments', data.totals.comments],
              ['Blank lines', data.totals.blank],
              ['Imports', data.totals.imports],
              ['Exports', data.totals.exports],
              ['Deepest folder depth', data.maxDepth],
              ['Deepest file', <span key="d" className="break-all text-xs">{data.deepestFile || '—'}</span>],
            ]}
          />
        </Card>
      </div>
      <Card>
        <h3 className="mb-3 text-sm font-medium text-slate-300">Largest files</h3>
        {(data.largestFiles ?? []).length === 0 ? (
          <Empty message="No files" />
        ) : (
          <Table
            headers={['File', 'Code lines', 'Complexity', 'Functions']}
            rows={data.largestFiles.map((f) => [
              <span key="p" className="text-sky-400">
                {f.path}
              </span>,
              f.linesCode,
              f.complexity,
              f.funcCount,
            ])}
          />
        )}
      </Card>
      <Card>
        <h3 className="mb-3 text-sm font-medium text-slate-300">Most complex files</h3>
        {(data.mostComplexFiles ?? []).length === 0 ? (
          <Empty message="No files" />
        ) : (
          <Table
            headers={['File', 'Complexity', 'Functions', 'Max nesting']}
            rows={data.mostComplexFiles.map((f) => [
              <span key="p" className="text-sky-400">
                {f.path}
              </span>,
              f.complexity,
              f.funcCount,
              f.maxNesting,
            ])}
          />
        )}
      </Card>
    </div>
  )
}

export default function Metrics() {
  return (
    <RepoProvider>
      <MetricsPage />
    </RepoProvider>
  )
}
