import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Bug, Copy, RotateCcw } from 'lucide-react'
import { Component, type ErrorInfo, type ReactNode } from 'react'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { EmptyState } from './EmptyState.tsx'
import { reportBug } from './reportBug.ts'

function Fallback({ error, stack, onRetry }: { error: Error; stack: string; onRetry: () => void }) {
  const { t } = useLingui()
  const copy = () => {
    Clipboard.SetText(`${error.stack ?? error.message}\n${stack}`)
      .then(() => useToasts.getState().push({ kind: 'success', title: t`Details copied` }))
      .catch(reportUnexpected)
  }
  return (
    <EmptyState
      icon={<Bug size={40} aria-hidden={true} />}
      title={t`Something went wrong in this view`}
      action={
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button variant="contained" startIcon={<RotateCcw size={16} />} onClick={onRetry}>
            {t`Retry`}
          </Button>
          <Button startIcon={<Copy size={16} />} onClick={copy}>
            {t`Copy details`}
          </Button>
          <Button variant="text" onClick={() => reportBug('')}>
            {t`Report a bug`}
          </Button>
        </Box>
      }
    >
      {t`The rest of Mortar still works. Try again, or copy the details for a bug report.`}
    </EmptyState>
  )
}

interface Props {
  children: ReactNode
  // A change of resetKey (the tab or profile shown) clears the error so the new view gets a fresh start.
  resetKey?: string
}

interface State {
  error: Error | null
  stack: string
  key: string | undefined
}

// ErrorBoundary keeps a render error in one view from blanking the whole window.
export class ErrorBoundary extends Component<Props, State> {
  override state: State = { error: null, stack: '', key: this.props.resetKey }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { error }
  }

  static getDerivedStateFromProps(props: Props, state: State): Partial<State> | null {
    return props.resetKey === state.key ? null : { error: null, stack: '', key: props.resetKey }
  }

  override componentDidCatch(_error: Error, info: ErrorInfo) {
    this.setState({ stack: info.componentStack ?? '' })
  }

  override render() {
    const { error, stack } = this.state
    if (error) {
      return (
        <Fallback
          error={error}
          stack={stack}
          onRetry={() => this.setState({ error: null, stack: '' })}
        />
      )
    }
    return this.props.children
  }
}
