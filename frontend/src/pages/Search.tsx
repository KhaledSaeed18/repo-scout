import { useState } from 'react'
import { Badge, Button, Card, Empty, Input, Select, Spinner } from '@/components/ui'
import RepoSelector from '../components/RepoSelector'
import { api, useRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'
import type { SearchResult } from '../lib/types'

const modes = [
  { id: 'content', label: 'Content' },
  { id: 'regex', label: 'Regex' },
  { id: 'filename', label: 'Filename' },
  { id: 'folder', label: 'Folder' },
  { id: 'extension', label: 'Extension' },
]

function SearchPage() {
  const { repoId } = useRepoContext()
  const repo = useRepo(repoId).data
  const [query, setQuery] = useState('')
  const [mode, setMode] = useState('content')
  const [caseSensitive, setCaseSensitive] = useState(false)
  const [wholeWord, setWholeWord] = useState(false)
  const [result, setResult] = useState<SearchResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const run = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!query.trim() || repoId === 0) return
    setLoading(true)
    setError('')
    try {
      setResult(await api.search({ repo: repoId, query, mode, case: caseSensitive, word: wholeWord }))
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Search</h1>
        <RepoSelector />
      </div>
      <Card>
        <form onSubmit={run} className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search term or pattern"
              className="w-72"
            />
            <Select value={mode} onChange={(e) => setMode(e.target.value)}>
              {modes.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.label}
                </option>
              ))}
            </Select>
            <label className="flex items-center gap-1 text-sm text-slate-400">
              <input type="checkbox" checked={caseSensitive} onChange={(e) => setCaseSensitive(e.target.checked)} />
              Case
            </label>
            <label className="flex items-center gap-1 text-sm text-slate-400">
              <input type="checkbox" checked={wholeWord} onChange={(e) => setWholeWord(e.target.checked)} />
              Whole word
            </label>
            <Button type="submit" disabled={loading || !query.trim()}>
              {loading && <Spinner />}
              Search
            </Button>
          </div>
          {error && <span className="text-sm text-rose-400">{error}</span>}
        </form>
      </Card>

      {result &&
        (result.hits.length === 0 ? (
          <Empty message="No matches" />
        ) : (
          <Card>
            <div className="mb-3 flex items-center justify-between text-sm text-slate-400">
              <span>
                {result.total} hits{result.truncated ? ' (truncated)' : ''}
              </span>
            </div>
            <div className="flex flex-col gap-2">
              {result.hits.map((hit) => (
                <div key={hit.fileId} className="rounded border border-slate-800 p-2">
                  <div className="flex items-center gap-2 text-sm">
                    <span className="font-medium text-sky-400">{hit.path}</span>
                    {hit.language && <Badge>{hit.language}</Badge>}
                    <span className="ml-auto text-xs text-slate-500">{hit.linesTotal} lines</span>
                  </div>
                  <pre className="mt-1 overflow-x-auto text-xs text-slate-400">
                    {hit.matches
                      .slice(0, 5)
                      .map((m) => `${String(m.line).padStart(4)}  ${m.text}`)
                      .join('\n')}
                  </pre>
                </div>
              ))}
            </div>
          </Card>
        ))}
    </div>
  )
}

export default function Search() {
  return (
    <RepoProvider>
      <SearchPage />
    </RepoProvider>
  )
}
