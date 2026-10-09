import { TriangleAlert } from 'lucide-react'
import { Component, type ReactNode } from 'react'
import { Button } from '@/components/ui'

interface Props {
  children: ReactNode
  /** When this changes (for example the route), a caught error is cleared. */
  resetKey?: string
}

interface State {
  error: Error | null
  resetKey?: string
}

/** Contains a render crash so the rest of the app keeps working. */
export default class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null, resetKey: this.props.resetKey }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { error }
  }

  static getDerivedStateFromProps(props: Props, state: State): Partial<State> | null {
    return props.resetKey !== state.resetKey ? { error: null, resetKey: props.resetKey } : null
  }

  render() {
    const { error } = this.state
    if (!error) return this.props.children
    return (
      <div role="alert" className="mx-auto flex max-w-lg flex-col items-start gap-3 px-6 py-16">
        <TriangleAlert className="size-6 text-destructive" aria-hidden />
        <h1 className="text-xl font-semibold">This view stopped working</h1>
        <p className="text-sm text-muted-foreground">
          Something in the page failed while rendering. Your scan data is safe. Try again, or reload the app if it keeps
          happening.
        </p>
        <pre className="max-w-full overflow-x-auto rounded-sm border bg-card px-3 py-2 font-mono text-xs">{error.message}</pre>
        <div className="flex gap-2">
          <Button onClick={() => this.setState({ error: null })}>Try again</Button>
          <Button variant="outline" onClick={() => window.location.reload()}>
            Reload app
          </Button>
        </div>
      </div>
    )
  }
}
