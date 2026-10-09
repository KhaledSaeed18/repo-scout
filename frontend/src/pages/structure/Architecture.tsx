import { Download, Network } from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Button, buttonVariants } from '@/components/ui'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader, Section } from '@/components/layout'
import { Empty, QueryView } from '@/components/states'
import { api, useArchitecture } from '@/lib/api'
import type { Architecture, Repository } from '@/lib/types'
import ImportGraph from './ImportGraph'

const fileViewLimit = 400
type View = { kind: 'folders'; depth: number } | { kind: 'files' }

function PathList({ paths, link, limit = 12 }: { paths: string[]; link?: boolean; limit?: number }) {
  const [expanded, setExpanded] = useState(false)
  const shown = expanded ? paths : paths.slice(0, limit)
  return (
    <>
      <ul className="flex flex-col gap-1">
        {shown.map((p) => (
          <li key={p} className="truncate font-mono text-[0.8125rem]" title={p}>
            {link ? (
              <Link to={`/files?file=${encodeURIComponent(p)}`} className="hover:text-primary hover:underline">
                {p}
              </Link>
            ) : (
              p
            )}
          </li>
        ))}
      </ul>
      {paths.length > limit && (
        <Button variant="link" size="sm" className="mt-1 px-0" onClick={() => setExpanded((v) => !v)}>
          {expanded ? 'Show fewer' : `Show all ${paths.length}`}
        </Button>
      )}
    </>
  )
}

function Finding({ title, description, count, children }: { title: string; description: string; count: number; children: ReactNode }) {
  return (
    <Section title={`${title} (${count})`} description={description} className="lg:mt-0">
      {count === 0 ? <p className="text-sm text-muted-foreground">None found.</p> : children}
    </Section>
  )
}

function ViewPicker({ view, setView, fileCount }: { view: View; setView: (v: View) => void; fileCount: number }) {
  const options: { label: string; view: View }[] = [
    { label: 'Top folders', view: { kind: 'folders', depth: 1 } },
    { label: 'Two levels', view: { kind: 'folders', depth: 2 } },
    { label: 'Three levels', view: { kind: 'folders', depth: 3 } },
    { label: 'Files', view: { kind: 'files' } },
  ]
  return (
    <div role="radiogroup" aria-label="Graph detail" className="flex flex-wrap rounded-md border bg-card p-0.5">
      {options.map((o) => {
        const active = o.view.kind === view.kind && (o.view.kind === 'files' || (view.kind === 'folders' && o.view.depth === view.depth))
        const disabled = o.view.kind === 'files' && fileCount > fileViewLimit
        return (
          <button
            key={o.label}
            role="radio"
            aria-checked={active}
            disabled={disabled}
            title={disabled ? `Too many files to draw (${fileCount}); use a folder view` : undefined}
            onClick={() => setView(o.view)}
            className={
              'cursor-pointer rounded-sm px-3 py-1 text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-40 ' +
              (active ? 'bg-accent font-medium text-accent-foreground' : 'text-muted-foreground hover:text-foreground')
            }
          >
            {o.label}
          </button>
        )
      })}
    </div>
  )
}

function Report({ repo, data }: { repo: Repository; data: Architecture }) {
  const [view, setView] = useState<View>({ kind: 'folders', depth: 2 })
  const folders = useMemo(() => new Set(data.folders), [data.folders])
  const fileCount = useMemo(() => new Set(data.edges.flatMap((e) => [e.from, e.to])).size, [data.edges])

  if (data.edges.length === 0) {
    return (
      <Empty icon={Network} title="No imports between files were found">
        Import analysis covers Go, TypeScript, JavaScript, Python, Rust, Java, Kotlin, C#, C, C++, PHP and Swift.
      </Empty>
    )
  }
  return (
    <>
      <Section title="Import graph" actions={<ViewPicker view={view} setView={setView} fileCount={fileCount} />}>
        <ImportGraph key={`${repo.id}-${view.kind}-${view.kind === 'folders' ? view.depth : 0}`} edges={data.edges} folders={folders} depth={view.kind === 'folders' ? view.depth : undefined} />
      </Section>
      <div className="mt-10 grid grid-cols-1 gap-10 lg:grid-cols-2">
        <Finding
          title="Circular dependencies"
          description="Folders that import each other in a loop. Cycles make code harder to change in isolation."
          count={data.cycles.length}
        >
          <ol className="flex flex-col gap-3">
            {data.cycles.map((c) => (
              <li key={c.join()} className="rounded-sm border bg-card p-3 font-mono text-[0.8125rem] break-all">
                {[...c, c[0]].join(' → ')}
              </li>
            ))}
          </ol>
        </Finding>
        <Finding
          title="Entry points"
          description="Files named like program entries, such as main.go or index.ts. Reachability starts here."
          count={data.entryPoints.length}
        >
          <PathList paths={data.entryPoints} link />
        </Finding>
        <Finding
          title="Possibly unused files"
          description="Source files no entry point reaches through imports. Tests, scripts and dynamic imports show up here too, so check before deleting."
          count={data.deadFiles.length}
        >
          <PathList paths={data.deadFiles} link />
        </Finding>
        <Finding
          title="Folders nothing imports"
          description="Source folders without an entry point that no other folder imports."
          count={data.unusedModules.length}
        >
          <PathList paths={data.unusedModules} />
        </Finding>
      </div>
    </>
  )
}

function ArchitecturePage({ repo }: { repo: Repository }) {
  const q = useArchitecture(repo.id)
  return (
    <>
      <PageHeader
        title="Architecture"
        description={`How the source files and folders of ${repo.name} import each other.`}
        actions={
          <a href={api.svgUrl(repo.id)} download className={buttonVariants({ variant: 'outline', size: 'sm' })}>
            <Download />
            SVG
          </a>
        }
      />
      <QueryView query={q} label="architecture">
        {(data) => <Report repo={repo} data={data} />}
      </QueryView>
    </>
  )
}

export default function ArchitectureRoute() {
  return <RequireRepo>{(repo) => <ArchitecturePage repo={repo} />}</RequireRepo>
}
