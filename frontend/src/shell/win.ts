import { Window } from '@wailsio/runtime'

const ignore = () => undefined

export const win = {
  minimise: () => Window.Minimise().catch(ignore),
  toggleMaximise: () => Window.ToggleMaximise().catch(ignore),
  close: () => Window.Close().catch(ignore),
  reportMaximised: (report: (maximised: boolean) => void): void => {
    Window.IsMaximised().then(report, ignore)
  },
}
