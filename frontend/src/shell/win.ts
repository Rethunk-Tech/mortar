import { msg } from '@lingui/core/macro'
import { Window } from '@wailsio/runtime'
import { i18n } from '../i18n/index.ts'
import { errorText } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const reportFailure = (title: string) => (err: unknown) => {
  const body = errorText(err)
  useToasts.getState().push({ kind: 'error', title, ...(body ? { body } : {}) })
}

export const win = {
  minimise: () => Window.Minimise().catch(reportFailure(i18n._(msg`Couldn't minimise the window`))),
  toggleMaximise: () =>
    Window.ToggleMaximise().catch(reportFailure(i18n._(msg`Couldn't resize the window`))),
  close: () => Window.Close().catch(reportFailure(i18n._(msg`Couldn't close the window`))),
  reportMaximised: (report: (maximised: boolean) => void): void => {
    Window.IsMaximised().then(report, reportFailure(i18n._(msg`Couldn't read the window state`)))
  },
}
