import { useLingui } from '@lingui/react/macro'
import { useQuitPrompt } from './quit.ts'
import { ConfirmationDialog } from './shell/dialog/ConfirmationDialog.tsx'

export function QuitPrompt() {
  const message = useQuitPrompt((s) => s.message)
  const resolve = useQuitPrompt((s) => s.resolve)
  const { t } = useLingui()
  if (!(message && resolve)) {
    return null
  }
  const answer = (confirmed: boolean) => {
    useQuitPrompt.setState({ message: '', resolve: null })
    resolve(confirmed)
  }
  return (
    <ConfirmationDialog
      open={true}
      title={t`Quit Mortar?`}
      body={message}
      confirmLabel={t`Quit`}
      onCancel={() => answer(false)}
      onConfirm={() => answer(true)}
    />
  )
}
