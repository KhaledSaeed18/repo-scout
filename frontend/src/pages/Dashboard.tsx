import { Link } from 'react-router-dom'
import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import {
  Badge,
  Button,
  Card,
  Empty,
  Spinner,
  Stat,
  Table,
} from '../components/ui'
import RepoSelector from '../components/RepoSelector'
import { api, useDeleteRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'
import type { Repository } from '../lib/types'
function formatBytes(n: number) {
  if (n > 1 << 30) return `${(n / (1 << 30)).toFixed(1)} GiB`
  if (n > 1 << 20) return `${(n / (1 << 20)).toFixed(1)} MiB`
  return `${(n / 1024).toFixed(1)} KiB`
}

function statusColor(s: string) {
  switch (s) {
    case 'ready':
      return 'bg-emerald-500/20 text-emerald-400 border-emerald-800'
    case 'scanning':
      return 'bg-amber-500/20 text-amber-400 border-amber-800'
    default:
      return 'bg-rose-500/20 text-rose-400 border-rose-800'
  }
}

function LanguageBreakdown({ repo }: { repo: Repository }) {
  if (repo.fileCount === 0) return <Empty message="No files scanned yet" />
  const data = [
    { name: 'Code', value: repo.totalCode },
    { name: 'Comments', value: repo.totalComments },
    { name: 'Blank', value: repo.totalBlank },
  ]
  const colors = ['#38bdf8', '#f59e0b', '#64748b']
  return (
    <Card>
      <h3 className="mb-3 text-sm font-medium text-slate-300">Line composition</h3>
      <ResponsiveContainer width="100%" height={220}>
        <BarChart data={data}>
          <XAxis dataKey="name" stroke="#64748b" fontSize={12} />
          <YAxis stroke="#64748b" fontSize={12} />
          <Tooltip
            contentStyle={{ background: '#0f172a', border: '1px solid #334155' }}
            labelStyle={{ color: '#e2e8f0' }}
          />
          <Bar dataKey="value" radius={[4, 4, 0, 0]}>
            {data.map((_, i) => (
              <Cell key={i} fill={colors[i]} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </Card>
  )
}

function RepoCard({ repo }: { repo: Repository }) {
  const del = useDeleteRepo()
  return (
    <Card className="flex flex-col gap-3">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="truncate font-medium text-slate-100">{repo.name}</h3>
          <p className="truncate text-xs text-slate-500">{repo.path}</p>
        </div>
        <Badge className={statusColor(repo.status)}>{repo.status}</Badge>
      </div>
      <div className="grid grid-cols-2 gap-2 text-sm">
        <span className="text-slate-400">
          Files <b className="text-slate-100">{repo.fileCount}</b>
        </span>
        <span className="text-slate-400">
          Commits <b className="text-slate-100">{repo.commitCount}</b>
        </span>
        <span className="text-slate-400">
          Contributors <b className="text-slate-100">{repo.contributorCount}</b>
        </span>
        <span className="text-slate-400">
          Size <b className="text-slate-100">{formatBytes(repo.totalSize)}</b>
        </span>
      </div>
      <div className="flex flex-wrap gap-2">
        <Link to={`/git?repo=${repo.id}`}>
          <Button variant="outline" className="text-xs">
            Git
          </Button>
        </Link>
        <Link to={`/architecture?repo=${repo.id}`}>
          <Button variant="outline" className="text-xs">
            Graph
          </Button>
        </Link>
        <a download href={api.exportUrl(repo.id, 'files', 'json')}>
          <Button variant="ghost" className="text-xs">
            JSON
          </Button>
        </a>
        <a download href={api.exportUrl(repo.id, 'files', 'csv')}>
          <Button variant="ghost" className="text-xs">
            CSV
          </Button>
        </a>
        <Button
          variant="ghost"
          className="ml-auto text-xs text-rose-400"
          onClick={() => {
            if (window.confirm(`Delete repository "${repo.name}" and its data?`)) {
              del.mutate(repo.id)
            }
          }}
        >
          Delete
        </Button>
      </div>
    </Card>
  )
}

function DashboardInner() {
  const { repos, repoId } = useRepoContext()
  if (!repos) return <Spinner />
  const repo = repos.repositories.find((r) => r.id === repoId)

  if (repos.repositories.length === 0) {
    return (
      <Empty message="No repositories scanned yet. Go to Scan to add one." />
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Dashboard</h1>
        <div className="flex items-center gap-2">
          <RepoSelector />
          <Link to="/scan">
            <Button>+ Scan repository</Button>
          </Link>
        </div>
      </div>

      {repo && (
        <>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            <Stat label="Files" value={repo.fileCount} />
            <Stat label="Code lines" value={repo.totalCode.toLocaleString()} />
            <Stat label="Commits" value={repo.commitCount.toLocaleString()} />
            <Stat
              label="Dependencies"
              value={repo.dependencyCount}
              sub={`${repo.dupGroupCount} duplicate groups`}
            />
          </div>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <LanguageBreakdown repo={repo} />
            <RepoCard repo={repo} />
          </div>
        </>
      )}

      <Card>
        <h2 className="mb-3 text-sm font-medium text-slate-300">Repositories</h2>
        <Table
          headers={['Name', 'Status', 'Files', 'LOC', 'Commits', 'Path']}
          rows={repos.repositories.map((r) => [
            <Link key={r.id} to={`/files?repo=${r.id}`} className="text-sky-400">
              {r.name}
            </Link>,
            <Badge key="s" className={statusColor(r.status)}>
              {r.status}
            </Badge>,
            r.fileCount,
            r.totalCode.toLocaleString(),
            r.commitCount,
            <span key="p" className="text-xs text-slate-500">
              {r.path}
            </span>,
          ])}
        />
      </Card>
    </div>
  )
}

export default function Dashboard() {
  return (
    <RepoProvider>
      <DashboardInner />
    </RepoProvider>
  )
}
