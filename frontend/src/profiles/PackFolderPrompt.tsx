import { useLingui } from '@lingui/react/macro'
import { Button, Typography } from '@mui/material'
import { useState } from 'react'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { TypedFolder } from '../shell/TypedFolder.tsx'
import { type InlineError, inlineError } from '../toasts/report.ts'

// Lets the player point Mortar at an r2modman or Gale data folder that is not in its default place.
export function PackFolderPrompt({ onChosen }: { onChosen: () => void }) {
  const { t } = useLingui()
  const [error, setError] = useState<InlineError | null>(null)
  const [typing, setTyping] = useState(false)
  const apply = async (dir: string) => {
    try {
      await SetByKey('packFolder', dir, '')
    } catch (e) {
      setError(inlineError(e))
      return
    }
    setError(null)
    onChosen()
  }
  // Without a native folder dialog (server mode) the path is typed instead.
  const choose = () => {
    PickFolder(t`r2modman or Gale data folder`)
      .then((dir) => (dir ? apply(dir) : undefined))
      .catch(() => setTyping(true))
  }
  return (
    <Typography variant="body2" color="text.secondary">
      {t`r2modman or Gale in another folder?`}{' '}
      <Button size="small" onClick={choose}>
        {t`Choose folder…`}
      </Button>
      {typing ? <TypedFolder label={t`r2modman or Gale data folder`} onUse={apply} /> : null}
      {error ? (
        <Typography
          component="span"
          variant="body2"
          color="error"
          role="alert"
          title={error.details}
          sx={{ ml: 1 }}
        >
          {error.message}
        </Typography>
      ) : null}
    </Typography>
  )
}
