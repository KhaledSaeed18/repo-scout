import { MousePointerClick } from 'lucide-react'
import { useSearchParams } from 'react-router-dom'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui'
import ExportLinks from '@/components/ExportLinks'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader } from '@/components/layout'
import { Empty } from '@/components/states'
import { useMetrics } from '@/lib/api'
import { plural } from '@/lib/format'
import { rankLanguages } from '@/lib/languages'
import type { Repository } from '@/lib/types'
import FileDetails from './FileDetails'
import FileTable from './FileTable'
import FileTree from './FileTree'

function FilesPage({ repo }: { repo: Repository }) {
  const [params, setParams] = useSearchParams()
  const selected = params.get('file') ?? ''
  const view = params.get('view') === 'table' ? 'table' : 'tree'
  const metrics = useMetrics(repo.id, 6)
  const languages = metrics.data ? rankLanguages(metrics.data.languages) : []

  const update = (patch: Record<string, string>) => {
    const next = new URLSearchParams(params)
    for (const [k, v] of Object.entries(patch)) {
      if (v) next.set(k, v)
      else next.delete(k)
    }
    setParams(next, { replace: true })
  }
  const select = (path: string) => update({ file: path, view: 'tree' })

  return (
    <>
      <PageHeader
        title="Files"
        description={`${plural(repo.fileCount, 'file')} scanned in ${repo.name}. Ignored folders are configured in Settings.`}
        actions={<ExportLinks repoId={repo.id} kind="files" />}
      />
      <Tabs value={view} onValueChange={(v) => update({ view: v === 'table' ? 'table' : '' })}>
        <TabsList className="mb-4 h-10 w-full justify-start border-b">
          <TabsTrigger value="tree">Browse</TabsTrigger>
          <TabsTrigger value="table">All files</TabsTrigger>
        </TabsList>
        <TabsContent value="tree">
          <div className="grid grid-cols-1 gap-8 lg:grid-cols-[minmax(16rem,22rem)_minmax(0,1fr)]">
            <div className="max-h-[70vh] overflow-y-auto rounded-md border bg-card p-1.5">
              <FileTree key={repo.id} repoId={repo.id} selected={selected} onSelect={select} />
            </div>
            <div className="min-w-0">
              {selected ? (
                <FileDetails repoId={repo.id} path={selected} />
              ) : (
                <Empty icon={MousePointerClick} title="Pick a file">
                  Its size, complexity and Git history show up here.
                </Empty>
              )}
            </div>
          </div>
        </TabsContent>
        <TabsContent value="table">
          <FileTable repoId={repo.id} languages={languages} onSelect={select} />
        </TabsContent>
      </Tabs>
    </>
  )
}

export default function Files() {
  return <RequireRepo>{(repo) => <FilesPage repo={repo} />}</RequireRepo>
}
