import { CopyCheck } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader, Section } from '@/components/layout'
import { Empty, QueryView } from '@/components/states'
import { useDuplicates, useSettings } from '@/lib/api'
import { formatNumber, plural } from '@/lib/format'
import type { DuplicateGroup, Repository } from '@/lib/types'

const span = (b: { startLine: number; endLine: number }) => b.endLine - b.startLine + 1

function Group({ group, index }: { group: DuplicateGroup; index: number }) {
  const longest = Math.max(group.lines, ...group.blocks.map(span))
  return (
    <Section
      title={`Group ${index + 1}`}
      description={`Up to ${plural(longest, 'line')} repeated in ${plural(group.fileCount, 'file')}.`}
      actions={<Badge variant="outline">{Math.round(group.similarity * 100)}% similar</Badge>}
    >
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(16rem,22rem)]">
        <figure className="min-w-0">
          <figcaption className="mb-1.5 text-xs text-muted-foreground">Sample of the repeated code, normalized for comparison</figcaption>
          <pre className="max-h-64 overflow-auto rounded-sm border bg-card p-3 font-mono text-xs leading-5">
            <code>{group.fragment}</code>
          </pre>
        </figure>
        <div>
          <p className="mb-1.5 text-xs text-muted-foreground">Where it appears</p>
          <ul className="flex flex-col gap-2">
            {group.blocks.map((b) => (
              <li key={`${b.filePath}:${b.startLine}`} className="text-sm">
                <Link
                  to={`/files?file=${encodeURIComponent(b.filePath)}`}
                  className="block truncate font-mono text-[0.8125rem] text-primary hover:underline"
                  title={b.filePath}
                >
                  {b.filePath}
                </Link>
                <span className="text-xs text-muted-foreground">
                  Lines {formatNumber(b.startLine)} to {formatNumber(b.endLine)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </Section>
  )
}

function DuplicatesPage({ repo }: { repo: Repository }) {
  const q = useDuplicates(repo.id)
  const settings = useSettings()
  const threshold = settings.data
    ? `Matches are blocks of at least ${plural(settings.data.dupMinLines, 'line')} that are ${Math.round(settings.data.dupMinSimilarity * 100)}% or more alike.`
    : ''
  return (
    <>
      <PageHeader
        title="Duplicates"
        description={`Code repeated across files in ${repo.name}. ${threshold} Thresholds are set in Settings.`}
      />
      <QueryView query={q} label="duplicates">
        {({ groups }) =>
          groups.length === 0 ? (
            <Empty icon={CopyCheck} title="No duplicated code found">
              Nothing repeated above the thresholds. Lower them in Settings to look for smaller copies.
            </Empty>
          ) : (
            groups.map((g, i) => <Group key={g.id} group={g} index={i} />)
          )
        }
      </QueryView>
    </>
  )
}

export default function Duplicates() {
  return <RequireRepo>{(repo) => <DuplicatesPage repo={repo} />}</RequireRepo>
}
