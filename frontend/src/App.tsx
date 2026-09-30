import { SettingsDialog } from './settings/SettingsDialog.tsx'
import { AppFrame } from './shell/AppFrame.tsx'
import { ToastHost } from './toasts/ToastHost.tsx'

export function App() {
  return (
    <>
      <AppFrame>{null}</AppFrame>
      <ToastHost />
      <SettingsDialog />
    </>
  )
}
