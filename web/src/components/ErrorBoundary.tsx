import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  error: Error | null
}

/**
 * ErrorBoundary catches render-time exceptions anywhere in the tree below it
 * and shows a recovery UI instead of unmounting to a blank white screen.
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Surface for debugging; the UI shows a friendly message.
    console.error('Unhandled UI error:', error, info)
  }

  handleReload = () => {
    window.location.reload()
  }

  render() {
    if (this.state.error) {
      return (
        <div role="alert" className="min-h-screen flex items-center justify-center p-8">
          <div className="max-w-md text-center">
            <h1 className="text-xl font-semibold text-foreground">Something went wrong</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              An unexpected error occurred while rendering this page.
            </p>
            <pre className="mt-4 text-left text-xs text-destructive whitespace-pre-wrap break-words">
              {this.state.error.message}
            </pre>
            <button
              onClick={this.handleReload}
              className="mt-6 rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-primary/90"
            >
              Reload
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
