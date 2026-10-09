import { ArrowDown, FileCode2 } from 'lucide-react'
import { useState } from 'react'
import {
  Button,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import { Empty, QueryView } from '@/components/states'
import { useFiles } from '@/lib/api'
import { formatNumber, formatRelative } from '@/lib/format'
import { colorOf, type LanguageShare } from '@/lib/languages'
import { cn } from '@/lib/utils'

const pageSize = 100
type Sort = 'loc' | 'complexity' | 'name'
const allLanguages = 'all'

function SortHead({ label, value, sort, onSort, className }: { label: string; value: Sort; sort: Sort; onSort: (s: Sort) => void; className?: string }) {
  return (
    <TableHead className={className} aria-sort={sort === value ? (value === 'name' ? 'ascending' : 'descending') : undefined}>
      <button
        onClick={() => onSort(value)}
        className={cn('inline-flex cursor-pointer items-center gap-1 hover:text-foreground', sort === value && 'text-foreground')}
      >
        {label}
        {sort === value && <ArrowDown className={cn('size-3', value === 'name' && 'rotate-180')} aria-hidden />}
      </button>
    </TableHead>
  )
}

/** Every file with its metrics, sorted and paged on the server. */
export default function FileTable({
  repoId,
  languages,
  onSelect,
}: {
  repoId: number
  languages: LanguageShare[]
  onSelect: (path: string) => void
}) {
  const [sort, setSort] = useState<Sort>('loc')
  const [language, setLanguage] = useState(allLanguages)
  const [page, setPage] = useState(0)
  const params: Record<string, string | number> = { sort, limit: pageSize, offset: page * pageSize }
  if (language !== allLanguages) params.language = language
  const q = useFiles(repoId, params)

  const onSort = (s: Sort) => {
    setSort(s)
    setPage(0)
  }
  const named = languages.filter((l) => l.name !== 'Other')
  const languageItems = [{ value: allLanguages, label: 'All languages' }, ...named.map((l) => ({ value: l.name, label: l.name }))]

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <Select
          items={languageItems}
          value={language}
          onValueChange={(v) => {
            setLanguage(v ?? allLanguages)
            setPage(0)
          }}
        >
          <SelectTrigger aria-label="Language" className="w-48">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {languageItems.map((l) => (
              <SelectItem key={l.value} value={l.value}>
                {l.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <QueryView query={q} label="files">
        {({ files, total }) =>
          files.length === 0 ? (
            <Empty icon={FileCode2} title="No files match" />
          ) : (
            <>
              <Table className={cn(q.isPlaceholderData && 'opacity-60')}>
                <TableHeader>
                  <TableRow>
                    <SortHead label="File" value="name" sort={sort} onSort={onSort} />
                    <TableHead>Language</TableHead>
                    <SortHead label="Lines of code" value="loc" sort={sort} onSort={onSort} className="text-right" />
                    <SortHead label="Complexity" value="complexity" sort={sort} onSort={onSort} className="text-right" />
                    <TableHead className="text-right">Functions</TableHead>
                    <TableHead className="text-right">Commits</TableHead>
                    <TableHead>Last change</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {files.map((f) => (
                    <TableRow key={f.path}>
                      <TableCell className="max-w-96">
                        <button
                          onClick={() => onSelect(f.path)}
                          className="block max-w-full cursor-pointer truncate text-left font-mono text-[0.8125rem] text-primary hover:underline"
                          title={f.path}
                        >
                          {f.path}
                        </button>
                      </TableCell>
                      <TableCell>
                        {f.language && (
                          <span className="inline-flex items-center gap-1.5">
                            <span className="size-2 rounded-[2px]" style={{ background: colorOf(languages, f.language) }} />
                            {f.language}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-right">{formatNumber(f.linesCode)}</TableCell>
                      <TableCell className="text-right">{formatNumber(f.complexity)}</TableCell>
                      <TableCell className="text-right">{formatNumber(f.funcCount)}</TableCell>
                      <TableCell className="text-right">{formatNumber(f.commits)}</TableCell>
                      <TableCell className="text-muted-foreground">
                        {f.lastCommitAt ? formatRelative(f.lastCommitAt) : 'Untracked'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <div className="flex items-center justify-between gap-3 text-sm text-muted-foreground">
                <span>
                  {formatNumber(page * pageSize + 1)} to {formatNumber(page * pageSize + files.length)} of {formatNumber(total)}
                </span>
                <div className="flex gap-1">
                  <Button variant="outline" size="sm" disabled={page === 0} onClick={() => setPage((p) => p - 1)}>
                    Previous
                  </Button>
                  <Button variant="outline" size="sm" disabled={(page + 1) * pageSize >= total} onClick={() => setPage((p) => p + 1)}>
                    Next
                  </Button>
                </div>
              </div>
            </>
          )
        }
      </QueryView>
    </div>
  )
}
