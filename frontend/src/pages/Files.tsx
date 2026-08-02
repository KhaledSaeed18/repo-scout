import { FileCode2, FolderGit2 } from 'lucide-react'
import { useState } from 'react'
import {
  Badge,
  Card,
  CardContent,
  EmptyState,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import RepoSelector from '../components/RepoSelector'
import { useFiles, useRepo, useTree } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'

const chartBadgeClasses = [
  'bg-chart-1/10 text-chart-1 dark:bg-chart-1/20',
  'bg-chart-2/10 text-chart-2 dark:bg-chart-2/20',
  'bg-chart-3/10 text-chart-3 dark:bg-chart-3/20',
  'bg-chart-4/10 text-chart-4 dark:bg-chart-4/20',
  'bg-chart-5/10 text-chart-5 dark:bg-chart-5/20',
]

const langChartIndex: Record<string, number> = {
  Go: 0,
  TypeScript: 1,
  JavaScript: 2,
  Python: 3,
  Rust: 4,
  Java: 0,
  'C++': 1,
}

function langColor(lang: string) {
  return chartBadgeClasses[langChartIndex[lang] ?? 4]
}

function FileTable({ repoId }: { repoId: number }) {
  const [sort, setSort] = useState('loc')
  const { data, isLoading } = useFiles(repoId, { sort, limit: 200 })
  if (isLoading) return <Spinner />
  const files = data?.files ?? []
  if (!files.length) return <EmptyState icon={FileCode2} title="No files" />
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <span>Sort by</span>
        <Select value={sort} onValueChange={(val) => val && setSort(val)}>
          <SelectTrigger className="w-48">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="loc">Code lines</SelectItem>
            <SelectItem value="complexity">Complexity</SelectItem>
            <SelectItem value="name">Name</SelectItem>
          </SelectContent>
        </Select>
        <span className="ml-auto">{data?.total} files</span>
      </div>
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Path</TableHead>
              <TableHead>Language</TableHead>
              <TableHead>Lines</TableHead>
              <TableHead>Code</TableHead>
              <TableHead>Complexity</TableHead>
              <TableHead>Funcs</TableHead>
              <TableHead>Max len</TableHead>
              <TableHead>Author</TableHead>
              <TableHead>Commits</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {files.map((f) => (
              <TableRow key={f.path}>
                <TableCell>{f.path}</TableCell>
                <TableCell>{f.language ? <Badge className={langColor(f.language)}>{f.language}</Badge> : ''}</TableCell>
                <TableCell>{f.linesTotal}</TableCell>
                <TableCell>{f.linesCode}</TableCell>
                <TableCell>{f.complexity}</TableCell>
                <TableCell>{f.funcCount}</TableCell>
                <TableCell>{f.maxFuncLen}</TableCell>
                <TableCell><span className="text-xs text-muted-foreground">{f.author || '—'}</span></TableCell>
                <TableCell>{f.commits}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

function Level({
  repoId,
  folder,
}: {
  repoId: number
  folder: string
}) {
  const { data } = useTree(repoId, folder)
  if (!data) return <Spinner />
  return (
    <ul className="ml-3 border-l border-border">
      {data.folders.map((sub) => {
        const path = folder ? `${folder}/${sub}` : sub
        return (
          <li key={path}>
            <FolderNode repoId={repoId} folder={path} label={sub} />
          </li>
        )
      })}
      {data.files.map((f) => (
        <li key={f.path}>
          <span className="flex items-center gap-1 px-2 py-1 text-sm text-muted-foreground">
            <span className="text-muted-foreground/70">▍</span>
            {f.name}
            <span className="ml-auto text-xs text-muted-foreground/70">{f.linesCode} loc</span>
          </span>
        </li>
      ))}
    </ul>
  )
}

function FolderNode({
  repoId,
  folder,
  label,
}: {
  repoId: number
  folder: string
  label: string
}) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-1 rounded px-2 py-1 text-left text-sm text-foreground hover:bg-muted/60"
      >
        <span className="text-muted-foreground">{open ? '▾' : '▸'}</span>
        <span className="text-muted-foreground">📁</span>
        {label}
      </button>
      {open && <Level repoId={repoId} folder={folder} />}
    </>
  )
}

function TreeView({ repoId }: { repoId: number }) {
  const { data, isLoading } = useTree(repoId, '')
  if (isLoading) return <Spinner />
  if (!data) return <EmptyState icon={FileCode2} title="No files" />
  return (
    <ul>
      {data.folders.map((sub) => (
        <li key={sub}>
          <FolderNode repoId={repoId} folder={sub} label={sub} />
        </li>
      ))}
      {data.files.map((f) => (
        <li key={f.path}>
          <span className="flex items-center gap-1 px-2 py-1 text-sm text-muted-foreground">
            <span className="text-muted-foreground/70">▍</span>
            {f.name}
            <span className="ml-auto text-xs text-muted-foreground/70">{f.linesCode} loc</span>
          </span>
        </li>
      ))}
    </ul>
  )
}

function FilesPage() {
  const { repoId } = useRepoContext()
  const repo = useRepo(repoId).data
  const [view, setView] = useState<'tree' | 'table'>('tree')
  if (repoId === 0) return <EmptyState icon={FolderGit2} title="Scan a repository first" />

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Files</h1>
        <div className="flex items-center gap-2">
          <Select value={view} onValueChange={(val) => setView(val as 'tree' | 'table')}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="tree">Tree</SelectItem>
              <SelectItem value="table">Table</SelectItem>
            </SelectContent>
          </Select>
          <RepoSelector />
        </div>
      </div>
      <Card>
        <CardContent>
          {view === 'tree' ? <TreeView repoId={repoId} /> : <FileTable repoId={repoId} />}
        </CardContent>
      </Card>
    </div>
  )
}

export default function Files() {
  return (
    <RepoProvider>
      <FilesPage />
    </RepoProvider>
  )
}
