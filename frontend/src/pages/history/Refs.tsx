import { GitBranch, Tag } from 'lucide-react'
import { Badge, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader, Section } from '@/components/layout'
import { Empty, QueryView } from '@/components/states'
import { useBranches, useTags } from '@/lib/api'
import { plural, shortHash } from '@/lib/format'
import type { Repository } from '@/lib/types'

function RefsPage({ repo }: { repo: Repository }) {
  const branches = useBranches(repo.id)
  const tags = useTags(repo.id)
  return (
    <>
      <PageHeader title="Branches & tags" description={`Local branches and tags in ${repo.name} at the last scan.`} />
      <Section title="Branches">
        <QueryView query={branches} label="branches">
          {(d) =>
            d.branches.length ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Branch</TableHead>
                    <TableHead>Points to</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {d.branches.map((b) => (
                    <TableRow key={b.name}>
                      <TableCell>
                        <span className="font-medium">{b.name}</span>
                        {b.isCurrent && (
                          <Badge variant="success" className="ml-2">
                            Checked out
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">{shortHash(b.commitHash)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <Empty icon={GitBranch} title="No local branches" />
            )
          }
        </QueryView>
      </Section>
      <Section title="Tags">
        <QueryView query={tags} label="tags">
          {(d) =>
            d.tags.length ? (
              <>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Tag</TableHead>
                      <TableHead>Points to</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {d.tags.map((t) => (
                      <TableRow key={t.name}>
                        <TableCell className="font-medium">{t.name}</TableCell>
                        <TableCell className="font-mono text-xs text-muted-foreground">{shortHash(t.commitHash)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                <p className="mt-3 text-xs text-muted-foreground">{plural(d.tags.length, 'tag')}</p>
              </>
            ) : (
              <Empty icon={Tag} title="No tags">
                Tags usually mark releases. Create one with <code className="font-mono">git tag v1.0.0</code>.
              </Empty>
            )
          }
        </QueryView>
      </Section>
    </>
  )
}

export default function Refs() {
  return <RequireRepo>{(repo) => <RefsPage repo={repo} />}</RequireRepo>
}
