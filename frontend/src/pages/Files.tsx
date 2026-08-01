import { useState } from 'react'
import { Badge, Card, Empty, Select, Spinner, Table } from '../components/ui'
import RepoSelector from '../components/RepoSelector'
import { useFiles, useRepo, useTree } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'

function langColor(lang: string) {
  const map: Record<string, string> = {
    Go: 'bg-sky-500/20 text-sky-300 border-sky-800',
    TypeScript: 'bg-blue-500/20 text-blue-300 border-blue-800',
    JavaScript: 'bg-yellow-500/20 text-yellow-300 border-yellow-800',
    Python: 'bg-emerald-500/20 text-emerald-300 border-emerald-800',
    Rust: 'bg-orange-500/20 text-orange-300 border-orange-800',
    Java: 'bg-red-500/20 text-red-300 border-red-800',
    'C++': 'bg-purple-500/20 text-purple-300 border-purple-800',
  }
  return map[lang] ?? 'bg-slate-500/20 text-slate-300 border-slate-700'
}

function FileTable({ repoId }: { repoId: number }) {
  const [sort, setSort] = useState('loc')
  const { data, isLoading } = useFiles(repoId, { sort, limit: 200 })
  if (isLoading) return <Spinner />
  const files = data?.files ?? []
  if (!files.length) return <Empty message="No files" />
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2 text-sm text-slate-400">
        <span>Sort by</span>
        <Select value={sort} onChange={(e) => setSort(e.target.value)}>
          <option value="loc">Code lines</option>
          <option value="complexity">Complexity</option>
          <option value="name">Name</option>
        </Select>
        <span className="ml-auto">{data?.total} files</span>
      </div>
      <Table
        headers={['Path', 'Language', 'Lines', 'Code', 'Complexity', 'Funcs', 'Max len', 'Author', 'Commits']}
        rows={files.map((f) => [
          f.path,
          f.language ? (
            <Badge key="l" className={langColor(f.language)}>
              {f.language}
            </Badge>
          ) : (
            ''
          ),
          f.linesTotal,
          f.linesCode,
          f.complexity,
          f.funcCount,
          f.maxFuncLen,
          <span key="a" className="text-xs text-slate-400">
            {f.author || '—'}
          </span>,
          f.commits,
        ])}
      />
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
    <ul className="ml-3 border-l border-slate-800">
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
          <span className="flex items-center gap-1 px-2 py-1 text-sm text-slate-400">
            <span className="text-slate-500">▍</span>
            {f.name}
            <span className="ml-auto text-xs text-slate-600">{f.linesCode} loc</span>
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
        className="flex w-full items-center gap-1 px-2 py-1 text-left text-sm text-slate-300 hover:bg-slate-800/60"
      >
        <span className="text-slate-500">{open ? '▾' : '▸'}</span>
        <span className="text-slate-400">📁</span>
        {label}
      </button>
      {open && <Level repoId={repoId} folder={folder} />}
    </>
  )
}

function TreeView({ repoId }: { repoId: number }) {
  const { data, isLoading } = useTree(repoId, '')
  if (isLoading) return <Spinner />
  if (!data) return <Empty message="No files" />
  return (
    <ul>
      {data.folders.map((sub) => (
        <li key={sub}>
          <FolderNode repoId={repoId} folder={sub} label={sub} />
        </li>
      ))}
      {data.files.map((f) => (
        <li key={f.path}>
          <span className="flex items-center gap-1 px-2 py-1 text-sm text-slate-400">
            <span className="text-slate-500">▍</span>
            {f.name}
            <span className="ml-auto text-xs text-slate-600">{f.linesCode} loc</span>
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
  if (repoId === 0) return <Empty message="Scan a repository first" />

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Files</h1>
        <div className="flex items-center gap-2">
          <Select value={view} onChange={(e) => setView(e.target.value as 'tree' | 'table')}>
            <option value="tree">Tree</option>
            <option value="table">Table</option>
          </Select>
          <RepoSelector />
        </div>
      </div>
      <Card>{view === 'tree' ? <TreeView repoId={repoId} /> : <FileTable repoId={repoId} />}</Card>
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
