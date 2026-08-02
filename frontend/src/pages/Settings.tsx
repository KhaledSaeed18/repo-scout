import { useEffect, useState } from 'react'
import { Button, Card, CardContent, Input, Spinner } from '@/components/ui'
import { useSaveSettings, useSettings } from '../lib/api'
import { RepoProvider } from '../lib/repoctx'
import type { Settings as SettingsType } from '../lib/types'

function SettingsPage() {
  const { data, isLoading } = useSettings()
  const save = useSaveSettings()
  const [form, setForm] = useState<SettingsType | null>(null)

  useEffect(() => {
    if (data) setForm(data)
  }, [data])

  if (isLoading || !form) return <Spinner />

  const set = <K extends keyof SettingsType>(key: K, value: SettingsType[K]) =>
    setForm((f) => (f ? { ...f, [key]: value } : f))

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-xl font-semibold">Settings</h1>
      <Card>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-1">
            <label className="text-sm text-muted-foreground">Ignored folders (comma separated)</label>
            <Input
              value={form.ignoreFolders.join(', ')}
              onChange={(e) =>
                set('ignoreFolders', e.target.value.split(',').map((s) => s.trim()).filter(Boolean))
              }
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-sm text-muted-foreground">Ignored extensions</label>
            <Input
              value={form.ignoreExtensions.join(', ')}
              onChange={(e) =>
                set('ignoreExtensions', e.target.value.split(',').map((s) => s.trim()).filter(Boolean))
              }
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-1">
              <label className="text-sm text-muted-foreground">Max file size (bytes)</label>
              <Input
                type="number"
                value={form.maxFileSize}
                onChange={(e) => set('maxFileSize', Number(e.target.value))}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm text-muted-foreground">Max folder depth (0 = unlimited)</label>
              <Input
                type="number"
                value={form.maxFileDepth}
                onChange={(e) => set('maxFileDepth', Number(e.target.value))}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm text-muted-foreground">Worker count</label>
              <Input
                type="number"
                value={form.workerCount}
                onChange={(e) => set('workerCount', Number(e.target.value))}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm text-muted-foreground">Max search files</label>
              <Input
                type="number"
                value={form.maxSearchFiles}
                onChange={(e) => set('maxSearchFiles', Number(e.target.value))}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm text-muted-foreground">Duplicate min lines</label>
              <Input
                type="number"
                value={form.dupMinLines}
                onChange={(e) => set('dupMinLines', Number(e.target.value))}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm text-muted-foreground">Duplicate min similarity</label>
              <Input
                type="number"
                step="0.05"
                value={form.dupMinSimilarity}
                onChange={(e) => set('dupMinSimilarity', Number(e.target.value))}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm text-muted-foreground">Theme</label>
              <Input value={form.theme} onChange={(e) => set('theme', e.target.value)} />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button onClick={() => save.mutate(form)} disabled={save.isPending}>
              {save.isPending && <Spinner />}
              Save settings
            </Button>
            {save.isSuccess && <span className="text-sm text-chart-2">Saved</span>}
            {save.isError && <span className="text-sm text-destructive">{save.error.message}</span>}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

export default function Settings() {
  return (
    <RepoProvider>
      <SettingsPage />
    </RepoProvider>
  )
}
