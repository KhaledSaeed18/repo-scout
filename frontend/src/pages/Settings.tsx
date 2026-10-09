import { Monitor, Moon, Sun } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Button, Input, Spinner } from '@/components/ui'
import TagInput from '@/components/TagInput'
import { PageHeader, Section } from '@/components/layout'
import { QueryView } from '@/components/states'
import { useSaveSettings, useSettings } from '@/lib/api'
import { applyTheme, isThemePreference, type ThemePreference } from '@/lib/theme'
import type { Settings as SettingsType } from '@/lib/types'
import { cn } from '@/lib/utils'

const mb = 1024 * 1024

function Field({ id, label, hint, children }: { id: string; label: string; hint?: ReactNode; children: ReactNode }) {
  return (
    <div className="grid grid-cols-1 gap-x-10 gap-y-2 border-b border-border/60 py-4 last:border-b-0 md:grid-cols-[minmax(0,18rem)_minmax(0,1fr)]">
      <div>
        <label htmlFor={id} className="font-medium">
          {label}
        </label>
        {hint && <p className="text-sm text-muted-foreground">{hint}</p>}
      </div>
      <div className="max-w-xl">{children}</div>
    </div>
  )
}

function NumberInput({
  id,
  value,
  onChange,
  min,
  max,
  step = 1,
  suffix,
}: {
  id: string
  value: number
  onChange: (n: number) => void
  min: number
  max?: number
  step?: number
  suffix?: string
}) {
  return (
    <div className="flex items-center gap-2">
      <Input
        id={id}
        type="number"
        inputMode="numeric"
        min={min}
        max={max}
        step={step}
        value={Number.isFinite(value) ? value : ''}
        onChange={(e) => onChange(e.target.value === '' ? NaN : Number(e.target.value))}
        className="w-32 tabular-nums"
      />
      {suffix && <span className="text-sm text-muted-foreground">{suffix}</span>}
    </div>
  )
}

const themes: { value: ThemePreference; label: string; icon: typeof Sun }[] = [
  { value: 'system', label: 'Match system', icon: Monitor },
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'dark', label: 'Dark', icon: Moon },
]

/** Mirrors the server's validation so mistakes show before saving. */
function problems(s: SettingsType): string[] {
  const out: string[] = []
  if (!Number.isInteger(s.workerCount) || s.workerCount < 1 || s.workerCount > 64) out.push('Parallel workers must be a whole number from 1 to 64.')
  if (!(s.maxFileSize > 0)) out.push('Largest file to read must be more than 0 MB.')
  if (!Number.isInteger(s.maxFileDepth) || s.maxFileDepth < 0) out.push('Folder depth must be 0 or more.')
  if (!Number.isInteger(s.maxSearchFiles) || s.maxSearchFiles < 1) out.push('Files to search must be at least 1.')
  if (!Number.isInteger(s.dupMinLines) || s.dupMinLines < 1) out.push('Duplicate size must be at least 1 line.')
  if (!(s.dupMinSimilarity > 0 && s.dupMinSimilarity <= 1)) out.push('Duplicate similarity must be above 0% and at most 100%.')
  return out
}

function SettingsForm({ saved }: { saved: SettingsType }) {
  const save = useSaveSettings()
  // Seeded once; later refetches (such as after a theme change) must not wipe
  // unsaved edits. The form resets only from its own successful save.
  const [form, setForm] = useState(saved)
  const set = <K extends keyof SettingsType>(key: K, value: SettingsType[K]) => setForm((f) => ({ ...f, [key]: value }))

  const dirty = JSON.stringify({ ...form, theme: '' }) !== JSON.stringify({ ...saved, theme: '' })
  const errors = problems(form)
  const theme = isThemePreference(saved.theme) ? saved.theme : 'system'

  const setTheme = (value: ThemePreference) => {
    applyTheme(value)
    save.mutate({ ...saved, theme: value })
  }

  return (
    <form
      // Steps only drive the spinner buttons; validity is checked by problems().
      noValidate
      onSubmit={(e) => {
        e.preventDefault()
        if (!errors.length) save.mutate({ ...form, theme }, { onSuccess: (next) => setForm(next) })
      }}
    >
      <Section title="Appearance">
        <Field id="theme" label="Theme" hint="Saved right away.">
          <div role="radiogroup" aria-label="Theme" className="inline-flex rounded-md border bg-card p-0.5">
            {themes.map((t) => (
              <button
                key={t.value}
                type="button"
                role="radio"
                aria-checked={theme === t.value}
                onClick={() => setTheme(t.value)}
                className={cn(
                  'inline-flex cursor-pointer items-center gap-1.5 rounded-sm px-3 py-1 text-sm transition-colors',
                  theme === t.value ? 'bg-accent font-medium text-accent-foreground' : 'text-muted-foreground hover:text-foreground',
                )}
              >
                <t.icon className="size-3.5" aria-hidden />
                {t.label}
              </button>
            ))}
          </div>
        </Field>
      </Section>

      <Section title="Scanning" description="Changes apply the next time a repository is scanned.">
        <Field id="ignore-folders" label="Folders to skip" hint="Any folder with one of these names is skipped, at any depth.">
          <TagInput
            id="ignore-folders"
            values={form.ignoreFolders}
            onChange={(v) => set('ignoreFolders', v)}
            placeholder="node_modules, dist"
          />
        </Field>
        <Field id="ignore-extensions" label="File types to skip" hint="Extensions such as log or min.js.">
          <TagInput
            id="ignore-extensions"
            values={form.ignoreExtensions}
            onChange={(v) => set('ignoreExtensions', v)}
            placeholder="log, lock"
            normalize={(v) => v.toLowerCase().replace(/^\.+/, '')}
          />
        </Field>
        <Field id="max-size" label="Largest file to read" hint="Bigger files are left out of every count.">
          <NumberInput
            id="max-size"
            value={Math.round((form.maxFileSize / mb) * 10) / 10}
            onChange={(n) => set('maxFileSize', Math.round(n * mb))}
            min={0.1}
            step={0.5}
            suffix="MB"
          />
        </Field>
        <Field id="max-depth" label="Folder depth" hint="How many folders deep to look. 0 means no limit.">
          <NumberInput id="max-depth" value={form.maxFileDepth} onChange={(n) => set('maxFileDepth', n)} min={0} suffix="levels" />
        </Field>
        <Field id="workers" label="Parallel workers" hint="Files read at the same time during a scan.">
          <NumberInput id="workers" value={form.workerCount} onChange={(n) => set('workerCount', n)} min={1} max={64} />
        </Field>
      </Section>

      <Section title="Search">
        <Field id="search-files" label="Files to search" hint="Text search stops after reading this many files.">
          <NumberInput id="search-files" value={form.maxSearchFiles} onChange={(n) => set('maxSearchFiles', n)} min={1} step={1000} suffix="files" />
        </Field>
      </Section>

      <Section title="Duplicate detection">
        <Field id="dup-lines" label="Smallest duplicate" hint="Shorter repeated blocks are ignored.">
          <NumberInput id="dup-lines" value={form.dupMinLines} onChange={(n) => set('dupMinLines', n)} min={1} suffix="lines" />
        </Field>
        <Field id="dup-similarity" label="Similarity" hint="How alike two blocks must be to count as copies.">
          <NumberInput
            id="dup-similarity"
            value={Math.round(form.dupMinSimilarity * 100)}
            onChange={(n) => set('dupMinSimilarity', n / 100)}
            min={1}
            max={100}
            step={5}
            suffix="%"
          />
        </Field>
      </Section>

      <div className="sticky bottom-0 mt-8 flex flex-wrap items-center gap-3 border-t bg-background/95 py-4 backdrop-blur-sm">
        <Button type="submit" disabled={!dirty || errors.length > 0 || save.isPending}>
          {save.isPending && <Spinner className="border-primary-foreground/40 border-t-primary-foreground" />}
          Save changes
        </Button>
        <Button type="button" variant="ghost" disabled={!dirty} onClick={() => setForm(saved)}>
          Discard changes
        </Button>
        <span role="status" className="text-sm">
          {errors[0] ? (
            <span className="text-destructive">{errors[0]}</span>
          ) : save.isError ? (
            <span className="text-destructive">{save.error.message}</span>
          ) : save.isSuccess && !dirty ? (
            <span className="text-success">Settings saved.</span>
          ) : dirty ? (
            <span className="text-muted-foreground">You have unsaved changes.</span>
          ) : null}
        </span>
      </div>
    </form>
  )
}

export default function Settings() {
  const q = useSettings()
  return (
    <>
      <PageHeader title="Settings" description="Preferences for every repository. They are stored with your scan data on this machine." />
      <QueryView query={q} label="settings">
        {(saved) => <SettingsForm saved={saved} />}
      </QueryView>
    </>
  )
}
