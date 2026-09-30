import { Window } from '@wailsio/runtime'

const logFailure = (action: string) => (err: unknown) => {
  console.error(`window: ${action} failed`, err)
}

export const win = {
  minimise: () => Window.Minimise().catch(logFailure('minimise')),
  toggleMaximise: () => Window.ToggleMaximise().catch(logFailure('toggle maximise')),
  close: () => Window.Close().catch(logFailure('close')),
  reportMaximised: (report: (maximised: boolean) => void): void => {
    Window.IsMaximised().then(report, logFailure('read maximised state'))
  },
}
