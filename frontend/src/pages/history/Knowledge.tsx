import { UserRoundCheck } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Badge, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import Meter from '@/components/Meter'
import OptionSelect from '@/components/OptionSelect'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader, Section } from '@/components/layout'
import { Empty, QueryView } from '@/components/states'
import { useKnowledge } from '@/lib/api'
import { formatDate, formatNumber, formatPercent, formatRelative, plural } from '@/lib/format'
import { describeBusFactor, folderDepths, folderLabel, inactiveWindows } from '@/lib/risk'
import type { KnowledgeReport, Repository } from '@/lib/types'

function Figure({ value, label, detail }: { value: string; label: string; detail: string }) {
  return (
    <div className="flex max-w-64 flex-col gap-0.5 px-5 py-3 first:pl-0">
      <span className="font-heading text-3xl leading-none font-semibold tabular-nums">{value}</span>
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="text-xs text-muted-foreground">{detail}</span>
    </div>
  )
}

function Summary({ r }: { r: KnowledgeReport }) {
  const active = r.owners.filter((o) => o.active).length
  const inactiveShare = r.lines ? r.inactiveLines / r.lines : 0
  return (
    <div className="mb-2 flex flex-wrap divide-x border-y">
      <Figure value={formatNumber(r.busFactor)} label="bus factor" detail={describeBusFactor(r.busFactor)} />
      <Figure
        value={formatPercent(inactiveShare)}
        label="of code has an inactive main author"
        detail={r.activeSince ? `Inactive means no commits since ${formatDate(r.activeSince)}.` : ''}
      />
      <Figure
        value={`${formatNumber(active)} of ${formatNumber(r.owners.length)}`}
        label="main authors still active"
        detail={`Across ${plural(r.files, 'file')} with history.`}
      />
    </div>
  )
}

function Folders({ r }: { r: KnowledgeReport }) {
  return (
    <Section
      title="By folder"
      description="Folders whose code mostly belongs to inactive authors come first, then the largest."
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Folder</TableHead>
            <TableHead className="text-right">Lines of code</TableHead>
            <TableHead className="text-right">Main authors</TableHead>
            <TableHead>Largest share</TableHead>
            <TableHead className="w-40">Inactive share</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {r.folders.map((f) => (
            <TableRow key={f.folder}>
              <TableCell className="max-w-72 truncate font-mono text-[0.8125rem]" title={folderLabel(f.folder)}>
                {folderLabel(f.folder)}
              </TableCell>
              <TableCell className="text-right">{formatNumber(f.lines)}</TableCell>
              <TableCell className="text-right">{formatNumber(f.owners)}</TableCell>
              <TableCell className="max-w-56">
                <span className="truncate">{f.topOwner}</span>{' '}
                <span className="text-muted-foreground">{formatPercent(f.topOwnerShare)}</span>
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <Meter value={f.inactiveShare} max={1} />
                  <span className="w-12 shrink-0 text-right text-xs text-muted-foreground">
                    {formatPercent(f.inactiveShare)}
                  </span>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Section>
  )
}

function AtRisk({ r }: { r: KnowledgeReport }) {
  return (
    <Section title="Files whose main author is inactive" description="Largest first. Someone else will need to learn these before changing them.">
      {r.atRisk.length === 0 ? (
        <p className="text-sm text-muted-foreground">Every file's main author is still committing.</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {r.atRisk.map((f) => (
            <li key={f.path} className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-0.5 text-sm">
              <Link
                to={`/files?file=${encodeURIComponent(f.path)}`}
                className="min-w-0 truncate font-mono text-[0.8125rem] text-primary hover:underline"
                title={f.path}
              >
                {f.path}
              </Link>
              <span className="shrink-0 text-muted-foreground">
                {f.owner} made {formatPercent(f.ownerShare)} of its commits, last committed {formatDate(f.ownerLastCommitAt)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Section>
  )
}

function Owners({ r }: { r: KnowledgeReport }) {
  const max = r.owners[0]?.lines ?? 0
  return (
    <Section title="Main authors" description="Lines of code in the files each person made the most commits to.">
      <ol className="grid max-w-3xl gap-3">
        {r.owners.map((o) => (
          <li key={o.email || o.author} className="grid grid-cols-[minmax(7rem,13rem)_1fr_auto] items-center gap-4 text-sm">
            <span className="flex min-w-0 items-center gap-2">
              <span className="truncate font-medium" title={o.email}>
                {o.author}
              </span>
              {!o.active && <Badge variant="outline">Inactive</Badge>}
            </span>
            <Meter value={o.lines} max={max} />
            <span className="w-40 text-right tabular-nums text-muted-foreground" title={`Last commit ${formatDate(o.lastCommitAt)}`}>
              {plural(o.files, 'file')}, {formatRelative(o.lastCommitAt)}
            </span>
          </li>
        ))}
      </ol>
    </Section>
  )
}

function KnowledgePage({ repo }: { repo: Repository }) {
  const [inactive, setInactive] = useState('6')
  const [depth, setDepth] = useState('2')
  const q = useKnowledge(repo.id, Number(depth), Number(inactive))
  return (
    <>
      <PageHeader
        title="Knowledge"
        description="Who knows which code. Each file belongs to the person who made the most commits to it, and people count as inactive when they have not committed for a while before the latest commit."
        actions={
          <>
            <OptionSelect label="Inactive after" value={inactive} options={inactiveWindows} onChange={setInactive} className="w-32" />
            <OptionSelect label="Group folders" value={depth} options={folderDepths} onChange={setDepth} className="w-36" />
          </>
        }
      />
      <QueryView query={q} label="knowledge">
        {(r) =>
          r.files === 0 ? (
            <Empty icon={UserRoundCheck} title="No files with history">
              Knowledge comes from Git history. Scan a folder that is a Git repository to see who wrote what; repositories scanned before this view existed need a new scan.
            </Empty>
          ) : (
            <>
              <Summary r={r} />
              <Folders r={r} />
              <AtRisk r={r} />
              <Owners r={r} />
            </>
          )
        }
      </QueryView>
    </>
  )
}

export default function Knowledge() {
  return <RequireRepo>{(repo) => <KnowledgePage repo={repo} />}</RequireRepo>
}
