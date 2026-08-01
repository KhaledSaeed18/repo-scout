import { Badge, Card, Empty, Spinner, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import RepoSelector from '../components/RepoSelector'
import { useDependencies, useRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'

const managerColor: Record<string, string> = {
  npm: 'bg-rose-500/20 text-rose-300 border-rose-800',
  composer: 'bg-purple-500/20 text-purple-300 border-purple-800',
  go: 'bg-sky-500/20 text-sky-300 border-sky-800',
  cargo: 'bg-orange-500/20 text-orange-300 border-orange-800',
  maven: 'bg-blue-500/20 text-blue-300 border-blue-800',
  pip: 'bg-emerald-500/20 text-emerald-300 border-emerald-800',
}

function DependenciesPage() {
  const { repoId } = useRepoContext()
  const repo = useRepo(repoId).data
  const { data, isLoading } = useDependencies(repoId)
  if (repoId === 0) return <Empty message="Scan a repository first" />
  if (isLoading) return <Spinner />
  const managers = data?.managers ?? []
  const deps = data?.dependencies ?? {}
  if (!managers.length) return <Empty message="No dependencies found" />

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Dependencies</h1>
        <RepoSelector />
      </div>
      <div className="flex flex-wrap gap-2">
        {managers.map((m) => (
          <Badge key={m} className={managerColor[m] ?? ''}>
            {m}: {deps[m].length}
          </Badge>
        ))}
      </div>
      {managers.map((m) => (
        <Card key={m} className="flex flex-col gap-3 overflow-hidden">
          <h2 className="text-sm font-medium text-slate-300">{m}</h2>
          <div className="overflow-x-auto">
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
                    <TableCell><span className="text-xs text-slate-500">{d.filePath}</span></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
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
