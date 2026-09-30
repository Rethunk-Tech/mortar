import { t } from '@lingui/core/macro'
import { Window } from '@wailsio/runtime'
import { useToasts } from '../toasts/store.ts'

const reportFailure = (title: string) => (err: unknown) => {
  const body = err instanceof Error ? err.message : typeof err === 'string' ? err : undefined
  useToasts.getState().push({ kind: 'error', title, ...(body ? { body } : {}) })
}

export const win = {
  minimise: () => Window.Minimise().catch(reportFailure(t`Couldn't minimise the window`)),
  toggleMaximise: () => Window.ToggleMaximise().catch(reportFailure(t`Couldn't resize the window`)),
  close: () => Window.Close().catch(reportFailure(t`Couldn't close the window`)),
  reportMaximised: (report: (maximised: boolean) => void): void => {
    Window.IsMaximised().then(report, reportFailure(t`Couldn't read the window state`))
  },
}
