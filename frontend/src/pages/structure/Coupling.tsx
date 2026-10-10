import { Link2 } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Badge, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import Meter from '@/components/Meter'
import OptionSelect from '@/components/OptionSelect'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader, Section } from '@/components/layout'
import { Empty, QueryView } from '@/components/states'
import { useCoupling } from '@/lib/api'
import { formatNumber, formatPercent, plural } from '@/lib/format'
import { couplingFilters, linkLabel } from '@/lib/risk'
import type { CouplingPair, Repository } from '@/lib/types'

function FileLink({ path }: { path: string }) {
  return (
    <Link
      to={`/files?file=${encodeURIComponent(path)}`}
      className="block truncate font-mono text-[0.8125rem] text-primary hover:underline"
      title={path}
    >
      {path}
    </Link>
  )
}

function Pairs({ pairs, hidden }: { pairs: CouplingPair[]; hidden: boolean }) {
  if (!pairs.length) {
    return (
      <Empty icon={Link2} title={hidden ? 'No hidden dependencies' : 'No files change together often'}>
        {hidden
          ? 'Every coupled pair is explained by an import, a shared package, a test or a lockfile.'
          : 'Files count as coupled after sharing at least three commits. Repositories scanned before this view existed need a new scan.'}
      </Empty>
    )
  }
  return (
    <Section
      title={hidden ? 'Hidden dependencies' : 'Coupled pairs'}
      description={`The ${plural(pairs.length, 'strongest pair')}. Commits that touch more than 50 files are left out.`}
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Files</TableHead>
            <TableHead className="text-right">Changed together</TableHead>
            <TableHead className="w-40">Coupling</TableHead>
            <TableHead>Why</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {pairs.map((p) => (
            <TableRow key={`${p.fileA}\n${p.fileB}`}>
              <TableCell className="max-w-[28rem]">
                <FileLink path={p.fileA} />
                <FileLink path={p.fileB} />
              </TableCell>
              <TableCell className="text-right" title={`${formatNumber(p.revisionsA)} and ${formatNumber(p.revisionsB)} commits in total`}>
                {plural(p.shared, 'commit')}
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <Meter value={p.degree} max={1} />
                  <span className="w-12 shrink-0 text-right text-xs text-muted-foreground">{formatPercent(p.degree)}</span>
                </div>
              </TableCell>
              <TableCell>
                <Badge variant={p.link ? 'outline' : 'warning'}>{linkLabel(p.link)}</Badge>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Section>
  )
}

function CouplingPage({ repo }: { repo: Repository }) {
  const [filter, setFilter] = useState('all')
  const hidden = filter === 'hidden'
  const q = useCoupling(repo.id, { hidden, limit: 200 })
  return (
    <>
      <PageHeader
        title="Change coupling"
        description="Files that keep changing in the same commits depend on each other, whatever the imports say. Pairs explained by an import, a shared Go package, a test or a lockfile are expected; the rest are hidden dependencies worth a look. Coupling is shared commits over the average of both files' commits."
        actions={<OptionSelect label="Show" value={filter} options={couplingFilters} onChange={setFilter} className="w-44" />}
      />
      <QueryView query={q} label="change coupling">
        {({ pairs }) => <Pairs pairs={pairs} hidden={hidden} />}
      </QueryView>
    </>
  )
}

export default function Coupling() {
  return <RequireRepo>{(repo) => <CouplingPage repo={repo} />}</RequireRepo>
}
