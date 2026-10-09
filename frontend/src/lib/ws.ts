import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { wsUrl } from './api'

/** Query keys that do not depend on scan results. */
const scanIndependent = new Set(['settings', 'browse'])

/**
 * Connects to the backend WebSocket and reconciles TanStack Query caches as
 * events arrive. Job events refresh job and repository status; repository
 * events refresh every scan-derived view, since a finished scan changes them
 * all. React Query dedupes overlapping refetches, so this stays cheap.
 */
export function useLiveUpdates() {
  const qc = useQueryClient()
  useEffect(() => {
    let closed = false
    let ws: WebSocket | null = null
    let timer: ReturnType<typeof setTimeout> | null = null

    const connect = () => {
      if (closed) return
      ws = new WebSocket(wsUrl())
      ws.onmessage = (event) => {
        let type: string
        try {
          type = (JSON.parse(event.data) as { type: string }).type
        } catch {
          return
        }
        if (type.startsWith('job.')) {
          void qc.invalidateQueries({ queryKey: ['jobs'] })
          void qc.invalidateQueries({ queryKey: ['repos'] })
        }
        if (type.startsWith('repository.')) {
          void qc.invalidateQueries({
            predicate: (q) => !scanIndependent.has(String(q.queryKey[0])),
          })
        }
      }
      ws.onclose = () => {
        if (!closed) timer = setTimeout(connect, 2000)
      }
      ws.onerror = () => ws?.close()
    }

    // Deferred so a StrictMode mount/unmount cycle never opens a socket.
    timer = setTimeout(connect, 0)
    return () => {
      closed = true
      if (timer) clearTimeout(timer)
      ws?.close()
    }
  }, [qc])
}
