import { Search as SearchIcon, SearchX } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Button, Input, Label } from '@/components/ui'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader } from '@/components/layout'
import { Empty, QueryView } from '@/components/states'
import { useSearch, type SearchParams } from '@/lib/api'
import { formatNumber, plural } from '@/lib/format'
import { highlight, matcherFor } from '@/lib/highlight'
import type { Repository, SearchHit } from '@/lib/types'
import { cn } from '@/lib/utils'

const modes = [
  { id: 'content', label: 'Text', hint: 'Find text inside files' },
  { id: 'regex', label: 'Regex', hint: 'Find a regular expression inside files' },
  { id: 'filename', label: 'File name', hint: 'Find files whose name contains the text' },
  { id: 'folder', label: 'Folder', hint: 'Find files in folders whose path contains the text' },
  { id: 'extension', label: 'Extension', hint: 'List files with an extension, such as ts' },
] as const

const contentModes = new Set(['content', 'regex'])
const maxLines = 6

function readParams(params: URLSearchParams): SearchParams {
  const mode = params.get('mode') ?? 'content'
  return {
    query: params.get('q') ?? '',
    mode: modes.some((m) => m.id === mode) ? mode : 'content',
    caseSensitive: params.get('case') === '1',
    wholeWord: params.get('word') === '1',
  }
}

function Hit({ hit, matcher }: { hit: SearchHit; matcher: RegExp | null }) {
  const shown = hit.matches.slice(0, maxLines)
  return (
    <li className="border-b border-border/70 py-3 last:border-b-0">
      <div className="flex items-baseline justify-between gap-4">
        <Link
          to={`/files?file=${encodeURIComponent(hit.path)}`}
          className="truncate font-mono text-[0.8125rem] font-medium text-primary hover:underline"
          title={hit.path}
        >
          {hit.path}
        </Link>
        <span className="shrink-0 text-xs text-muted-foreground">
          {hit.matches.length ? plural(hit.matches.length, 'match', 'matches') : hit.language || `${formatNumber(hit.linesTotal)} lines`}
        </span>
      </div>
      {shown.length > 0 && (
        <pre className="mt-2 overflow-x-auto rounded-sm border bg-card py-1.5 font-mono text-xs leading-5">
          {shown.map((m) => (
            <div key={m.line} className="flex">
              <span className="w-12 shrink-0 pr-3 text-right text-muted-foreground select-none">{m.line}</span>
              <code className="pr-3 whitespace-pre">
                {highlight(m.text, matcher).map((seg, i) =>
                  seg.match ? (
                    <mark key={i} className="rounded-[2px] bg-chart-3/30 text-foreground">
                      {seg.text}
                    </mark>
                  ) : (
                    <span key={i}>{seg.text}</span>
                  ),
                )}
              </code>
            </div>
          ))}
        </pre>
      )}
      {hit.matches.length > maxLines && (
        <p className="mt-1 text-xs text-muted-foreground">
          {plural(hit.matches.length - maxLines, 'more match', 'more matches')} in this file
        </p>
      )}
    </li>
  )
}

function Results({ repoId, params }: { repoId: number; params: SearchParams }) {
  const q = useSearch(repoId, params)
  const matcher = contentModes.has(params.mode)
    ? matcherFor({ query: params.query, regex: params.mode === 'regex', caseSensitive: params.caseSensitive, wholeWord: params.wholeWord })
    : null
  return (
    <QueryView query={q} label="results">
      {(r) =>
        r.hits.length === 0 ? (
          <Empty icon={SearchX} title={`Nothing matches “${params.query}”`}>
            Try another mode, turn off Match case or Whole word, or check the spelling.
          </Empty>
        ) : (
          <>
            <p className="mb-1 text-sm text-muted-foreground">
              {plural(r.hits.length, 'file')}
              {r.truncated && ' shown. More files match; refine the search to narrow them down.'}
            </p>
            <ol>
              {r.hits.map((hit) => (
                <Hit key={hit.fileId} hit={hit} matcher={matcher} />
              ))}
            </ol>
          </>
        )
      }
    </QueryView>
  )
}

function SearchPage({ repo }: { repo: Repository }) {
  const [urlParams, setUrlParams] = useSearchParams()
  const submitted = readParams(urlParams)
  const [draft, setDraft] = useState(submitted)
  useEffect(() => setDraft(readParams(urlParams)), [urlParams])

  const submit = (next: SearchParams) => {
    const p = new URLSearchParams()
    if (next.query.trim()) p.set('q', next.query.trim())
    if (next.mode !== 'content') p.set('mode', next.mode)
    if (next.caseSensitive) p.set('case', '1')
    if (next.wholeWord) p.set('word', '1')
    setUrlParams(p)
  }
  const isContent = contentModes.has(draft.mode)
  const mode = modes.find((m) => m.id === draft.mode)!

  return (
    <>
      <PageHeader title="Search" description={`Search the text and file names of ${repo.name} as of the last scan.`} />
      <form
        onSubmit={(e) => {
          e.preventDefault()
          submit(draft)
        }}
        className="mb-8 flex flex-col gap-3"
      >
        <div className="flex gap-2">
          <div className="relative flex-1">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={draft.query}
              onChange={(e) => setDraft({ ...draft, query: e.target.value })}
              placeholder={mode.hint}
              aria-label="Search for"
              className={cn('h-10 pl-9 text-base', draft.mode === 'regex' && 'font-mono text-sm')}
              autoFocus
              spellCheck={false}
            />
          </div>
          <Button type="submit" className="h-10 px-4" disabled={!draft.query.trim()}>
            Search
          </Button>
        </div>
        <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
          <div role="radiogroup" aria-label="Search in" className="flex rounded-md border bg-card p-0.5">
            {modes.map((m) => (
              <button
                key={m.id}
                type="button"
                role="radio"
                aria-checked={draft.mode === m.id}
                title={m.hint}
                onClick={() => setDraft({ ...draft, mode: m.id })}
                className={cn(
                  'cursor-pointer rounded-sm px-3 py-1 text-sm transition-colors',
                  draft.mode === m.id ? 'bg-accent font-medium text-accent-foreground' : 'text-muted-foreground hover:text-foreground',
                )}
              >
                {m.label}
              </button>
            ))}
          </div>
          {draft.mode !== 'extension' && (
            <Label className="font-normal text-muted-foreground">
              <input
                type="checkbox"
                className="size-4 accent-primary"
                checked={draft.caseSensitive}
                onChange={(e) => setDraft({ ...draft, caseSensitive: e.target.checked })}
              />
              Match case
            </Label>
          )}
          {isContent && (
            <Label className="font-normal text-muted-foreground">
              <input
                type="checkbox"
                className="size-4 accent-primary"
                checked={draft.wholeWord}
                onChange={(e) => setDraft({ ...draft, wholeWord: e.target.checked })}
              />
              Whole word
            </Label>
          )}
        </div>
      </form>
      {submitted.query ? (
        <Results repoId={repo.id} params={submitted} />
      ) : (
        <p className="text-sm text-muted-foreground">
          Results link to each file's details. Text search ignores files larger than the limit set in Settings.
        </p>
      )}
    </>
  )
}

export default function Search() {
  return <RequireRepo>{(repo) => <SearchPage repo={repo} />}</RequireRepo>
}
