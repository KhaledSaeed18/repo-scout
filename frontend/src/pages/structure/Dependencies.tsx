import { PackageSearch } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Badge, Input, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader, Section } from '@/components/layout'
import { Empty, QueryView } from '@/components/states'
import { useDependencies } from '@/lib/api'
import { plural } from '@/lib/format'
import type { Dependency, Repository } from '@/lib/types'

const ecosystems: Record<string, string> = {
  npm: 'npm',
  go: 'Go modules',
  pip: 'Python (pip)',
  cargo: 'Cargo',
  composer: 'Composer',
  maven: 'Maven',
}

const scopes: Record<string, { label: string; variant: 'default' | 'secondary' | 'outline' }> = {
  production: { label: 'Runtime', variant: 'default' },
  development: { label: 'Development', variant: 'secondary' },
  indirect: { label: 'Indirect', variant: 'outline' },
}

function scopeBadge(scope: string) {
  const s = scopes[scope] ?? { label: scope || 'Unspecified', variant: 'outline' as const }
  return <Badge variant={s.variant}>{s.label}</Badge>
}

function Manager({ name, deps }: { name: string; deps: Dependency[] }) {
  const manifests = new Set(deps.map((d) => d.filePath))
  const counts = deps.reduce<Record<string, number>>((acc, d) => ({ ...acc, [d.scope]: (acc[d.scope] ?? 0) + 1 }), {})
  const summary = Object.entries(counts)
    .map(([scope, n]) => `${n} ${(scopes[scope]?.label ?? scope).toLowerCase()}`)
    .join(', ')
  return (
    <Section
      title={ecosystems[name] ?? name}
      description={`${plural(deps.length, 'package')} (${summary}) from ${plural(manifests.size, 'manifest')}.`}
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Package</TableHead>
            <TableHead>Version</TableHead>
            <TableHead>Used for</TableHead>
            <TableHead>Declared in</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {deps.map((d) => (
            <TableRow key={`${d.filePath}:${d.name}:${d.scope}`}>
              <TableCell className="font-medium">{d.name}</TableCell>
              <TableCell className="font-mono text-xs">{d.version || '—'}</TableCell>
              <TableCell>{scopeBadge(d.scope)}</TableCell>
              <TableCell>
                <Link
                  to={`/files?file=${encodeURIComponent(d.filePath)}`}
                  className="font-mono text-xs text-muted-foreground hover:text-primary hover:underline"
                >
                  {d.filePath}
                </Link>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Section>
  )
}

function DependenciesPage({ repo }: { repo: Repository }) {
  const q = useDependencies(repo.id)
  const [filter, setFilter] = useState('')
  return (
    <>
      <PageHeader
        title="Dependencies"
        description={`Packages declared in the manifests of ${repo.name}, such as package.json, go.mod and requirements.txt.`}
        actions={
          <Input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Filter packages"
            aria-label="Filter packages"
            className="w-64"
          />
        }
      />
      <QueryView query={q} label="dependencies">
        {({ managers, dependencies }) => {
          if (!managers.length) {
            return (
              <Empty icon={PackageSearch} title="No manifests found">
                Repo Scout reads package.json, composer.json, go.mod, Cargo.toml, pom.xml and requirements.txt.
              </Empty>
            )
          }
          const needle = filter.trim().toLowerCase()
          const visible = managers
            .map((m) => [m, dependencies[m].filter((d) => !needle || d.name.toLowerCase().includes(needle))] as const)
            .filter(([, deps]) => deps.length)
          if (!visible.length) return <Empty icon={PackageSearch} title={`No packages match “${filter.trim()}”`} />
          return visible.map(([m, deps]) => <Manager key={m} name={m} deps={deps} />)
        }}
      </QueryView>
    </>
  )
}

export default function Dependencies() {
  return <RequireRepo>{(repo) => <DependenciesPage repo={repo} />}</RequireRepo>
}
