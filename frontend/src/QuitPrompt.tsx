import { useLingui } from '@lingui/react/macro'
import { useQuitPrompt } from './quit.ts'
import { ConfirmDialog } from './shell/ConfirmDialog.tsx'

export function QuitPrompt() {
  const { message, resolve } = useQuitPrompt()
  const { t } = useLingui()
  if (!(message && resolve)) {
    return null
  }
  const answer = (confirmed: boolean) => {
    useQuitPrompt.setState({ message: '', resolve: null })
    resolve(confirmed)
  }
  return (
    <ConfirmDialog
      open={true}
      title={t`Quit Mortar?`}
      body={message}
      confirmLabel={t`Quit`}
      onCancel={() => answer(false)}
      onConfirm={() => answer(true)}
    />
  )
}
