import { msg } from '@lingui/core/macro'
import { Window } from '@wailsio/runtime'
import { i18n } from '../i18n/index.ts'
import { reportError } from '../toasts/report.ts'

export const win = {
  minimise: () => Window.Minimise().catch(reportError(i18n._(msg`Could not minimise the window`))),
  toggleMaximise: () =>
    Window.ToggleMaximise().catch(reportError(i18n._(msg`Could not resize the window`))),
  close: () => Window.Close().catch(reportError(i18n._(msg`Could not close the window`))),
  reportMaximised: (report: (maximised: boolean) => void): void => {
    Window.IsMaximised().then(report, reportError(i18n._(msg`Could not read the window state`)))
  },
}
