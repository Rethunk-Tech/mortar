import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material'
import { useQuitPrompt } from './quit.ts'

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
    <Dialog open={true} onClose={() => answer(false)}>
      <DialogTitle>{t`Quit Mortar?`}</DialogTitle>
      <DialogContent>{message}</DialogContent>
      <DialogActions>
        <Button onClick={() => answer(false)}>{t`Cancel`}</Button>
        <Button onClick={() => answer(true)} autoFocus={true}>
          {t`Quit`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
