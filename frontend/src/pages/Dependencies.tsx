import { FolderGit2, PackageX } from 'lucide-react'
import { Badge, Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import RepoSelector from '../components/RepoSelector'
import { useDependencies, useRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'

const chartBadgeClasses = [
  'bg-chart-1/10 text-chart-1 dark:bg-chart-1/20',
  'bg-chart-2/10 text-chart-2 dark:bg-chart-2/20',
  'bg-chart-3/10 text-chart-3 dark:bg-chart-3/20',
  'bg-chart-4/10 text-chart-4 dark:bg-chart-4/20',
  'bg-chart-5/10 text-chart-5 dark:bg-chart-5/20',
]

const managerChartIndex: Record<string, number> = {
  npm: 3,
  composer: 1,
  go: 0,
  cargo: 4,
  maven: 2,
  pip: 0,
}

function managerColor(manager: string) {
  return chartBadgeClasses[managerChartIndex[manager] ?? 4]
}

function DependenciesPage() {
  const { repoId } = useRepoContext()
  const repo = useRepo(repoId).data
  const { data, isLoading } = useDependencies(repoId)
  if (repoId === 0) return <EmptyState icon={FolderGit2} title="Scan a repository first" />
  if (isLoading) return <Spinner />
  const managers = data?.managers ?? []
  const deps = data?.dependencies ?? {}
  if (!managers.length) return <EmptyState icon={PackageX} title="No dependencies found" />

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Dependencies</h1>
        <RepoSelector />
      </div>
      <div className="flex flex-wrap gap-2">
        {managers.map((m) => (
          <Badge key={m} className={managerColor(m)}>
            {m}: {deps[m].length}
          </Badge>
        ))}
      </div>
      {managers.map((m) => (
        <Card key={m}>
          <CardHeader>
            <CardTitle className="text-sm">{m}</CardTitle>
          </CardHeader>
          <CardContent className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Package</TableHead>
                  <TableHead>Version</TableHead>
                  <TableHead>Scope</TableHead>
                  <TableHead>Manifest</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {deps[m].map((d) => (
                  <TableRow key={d.name + d.version}>
                    <TableCell>{d.name}</TableCell>
                    <TableCell>{d.version}</TableCell>
                    <TableCell>{d.scope || '—'}</TableCell>
                    <TableCell><span className="text-xs text-muted-foreground">{d.filePath}</span></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

export default function Dependencies() {
  return (
    <RepoProvider>
      <DependenciesPage />
    </RepoProvider>
  )
}
