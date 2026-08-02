import { FolderGit2, ScanLine } from 'lucide-react'
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
  CardContent,
  CardHeader,
  CardTitle,
  EmptyState,
  Spinner,
  Stat,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import RepoSelector from '../components/RepoSelector'
import { api, useDeleteRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'
import type { Repository } from '../lib/types'
function formatBytes(n: number) {
  if (n > 1 << 30) return `${(n / (1 << 30)).toFixed(1)} GiB`
  if (n > 1 << 20) return `${(n / (1 << 20)).toFixed(1)} MiB`
  return `${(n / 1024).toFixed(1)} KiB`
}

function statusVariant(s: string): 'success' | 'warning' | 'destructive' {
  switch (s) {
    case 'ready':
      return 'success'
    case 'scanning':
      return 'warning'
    default:
      return 'destructive'
  }
}

function LanguageBreakdown({ repo }: { repo: Repository }) {
  if (repo.fileCount === 0) {
    return (
      <Card>
        <CardContent>
          <EmptyState icon={ScanLine} title="No files scanned yet" />
        </CardContent>
      </Card>
    )
  }
  const data = [
    { name: 'Code', value: repo.totalCode },
    { name: 'Comments', value: repo.totalComments },
    { name: 'Blank', value: repo.totalBlank },
  ]
  const colors = ['var(--chart-1)', 'var(--chart-3)', 'var(--chart-5)']
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Line composition</CardTitle>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={220}>
          <BarChart data={data}>
            <XAxis dataKey="name" stroke="var(--muted-foreground)" fontSize={12} />
            <YAxis stroke="var(--muted-foreground)" fontSize={12} />
            <Tooltip
              contentStyle={{ background: 'var(--popover)', border: '1px solid var(--border)' }}
              labelStyle={{ color: 'var(--popover-foreground)' }}
            />
            <Bar dataKey="value" radius={[4, 4, 0, 0]} isAnimationActive={false}>
              {data.map((_, i) => (
                <Cell key={i} fill={colors[i]} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}

function RepoCard({ repo }: { repo: Repository }) {
  const del = useDeleteRepo()
  return (
    <Card>
      <CardContent className="flex flex-col gap-3">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <h3 className="truncate font-medium text-foreground">{repo.name}</h3>
            <p className="truncate text-xs text-muted-foreground">{repo.path}</p>
          </div>
          <Badge variant={statusVariant(repo.status)}>{repo.status}</Badge>
        </div>
        <div className="grid grid-cols-2 gap-2 text-sm">
          <span className="text-muted-foreground">
            Files <b className="text-foreground">{repo.fileCount}</b>
          </span>
          <span className="text-muted-foreground">
            Commits <b className="text-foreground">{repo.commitCount}</b>
          </span>
          <span className="text-muted-foreground">
            Contributors <b className="text-foreground">{repo.contributorCount}</b>
          </span>
          <span className="text-muted-foreground">
            Size <b className="text-foreground">{formatBytes(repo.totalSize)}</b>
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
            className="ml-auto text-xs text-destructive"
            onClick={() => {
              if (window.confirm(`Delete repository "${repo.name}" and its data?`)) {
                del.mutate(repo.id)
              }
            }}
          >
            Delete
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function DashboardInner() {
  const { repos, repoId } = useRepoContext()
  if (!repos) return <Spinner />
  const repo = repos.repositories.find((r) => r.id === repoId)

  if (repos.repositories.length === 0) {
    return (
      <EmptyState
        icon={FolderGit2}
        title="No repositories scanned yet"
        description="Go to Scan to add your first repository."
      />
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
        <CardHeader>
          <CardTitle className="text-sm">Repositories</CardTitle>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Files</TableHead>
                <TableHead>LOC</TableHead>
                <TableHead>Commits</TableHead>
                <TableHead>Path</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {repos.repositories.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>
                    <Link to={`/files?repo=${r.id}`} className="text-primary">
                      {r.name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant(r.status)}>
                      {r.status}
                    </Badge>
                  </TableCell>
                  <TableCell>{r.fileCount}</TableCell>
                  <TableCell>{r.totalCode.toLocaleString()}</TableCell>
                  <TableCell>{r.commitCount}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{r.path}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
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
