import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { wsUrl } from './api'

/**
 * Connects to the backend WebSocket once and invalidates TanStack Query keys
 * whenever a job or repository event arrives. The reconciliation is cheap
 * because React Query dedupes overlapping refetches.
 */
export function useLiveUpdates() {
  const qc = useQueryClient()
  useEffect(() => {
    let closed = false
    let ws: WebSocket | null = null
    let retry: ReturnType<typeof setTimeout> | null = null

    const connect = () => {
      if (closed) return
      ws = new WebSocket(wsUrl())
      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data) as { type: string }
          if (msg.type.startsWith('job.')) {
            void qc.invalidateQueries({ queryKey: ['jobs'] })
          }
          if (msg.type.startsWith('repository.')) {
            void qc.invalidateQueries({ queryKey: ['repos'] })
            void qc.invalidateQueries({ queryKey: ['repo'] })
          }
        } catch {
          /* ignore malformed frames */
        }
      }
      ws.onclose = () => {
        if (!closed) retry = setTimeout(connect, 2000)
      }
      ws.onerror = () => ws?.close()
    }

    connect()
    return () => {
      closed = true
      if (retry) clearTimeout(retry)
      ws?.close()
    }
  }, [qc])
}
