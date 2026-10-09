import { X } from 'lucide-react'
import { useState } from 'react'

/** Editable list of short values. Enter or comma adds; Backspace on empty removes the last. */
export default function TagInput({
  id,
  values,
  onChange,
  placeholder,
  normalize = (v) => v,
}: {
  id: string
  values: string[]
  onChange: (values: string[]) => void
  placeholder?: string
  normalize?: (v: string) => string
}) {
  const [draft, setDraft] = useState('')
  const add = (raw: string) => {
    const next = raw
      .split(',')
      .map((v) => normalize(v.trim()))
      .filter((v) => v && !values.includes(v))
    if (next.length) onChange([...values, ...next])
    setDraft('')
  }
  return (
    <div className="flex min-h-9 flex-wrap items-center gap-1.5 rounded-md border border-input bg-card px-2 py-1.5 focus-within:border-ring focus-within:ring-2 focus-within:ring-ring/25">
      {values.map((v) => (
        <span key={v} className="inline-flex items-center gap-1 rounded-sm bg-secondary py-0.5 pr-1 pl-2 font-mono text-xs">
          {v}
          <button
            type="button"
            aria-label={`Remove ${v}`}
            onClick={() => onChange(values.filter((x) => x !== v))}
            className="cursor-pointer rounded-sm p-0.5 text-muted-foreground hover:bg-background hover:text-foreground"
          >
            <X className="size-3" />
          </button>
        </span>
      ))}
      <input
        id={id}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ',') {
            e.preventDefault()
            add(draft)
          } else if (e.key === 'Backspace' && !draft && values.length) {
            onChange(values.slice(0, -1))
          }
        }}
        onBlur={() => draft && add(draft)}
        placeholder={values.length ? undefined : placeholder}
        className="min-w-32 flex-1 bg-transparent py-0.5 font-mono text-xs outline-none placeholder:font-sans placeholder:text-sm placeholder:text-muted-foreground/80"
      />
    </div>
  )
}
