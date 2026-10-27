import { useLingui } from '@lingui/react/macro'
import { Button, Typography } from '@mui/material'
import { type ReactNode, useState } from 'react'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { TypedFolder } from '../shell/TypedFolder.tsx'
import { type InlineError, inlineError } from '../toasts/report.ts'

// Lets the player point Mortar at a mod manager's data folder that is not in its default place.
export function ManagerFolderPrompt({
  settingKey,
  label,
  note,
  rejected,
  onChosen,
}: {
  settingKey: string
  label: string
  note: ReactNode
  // What to say when the folder holds none of the manager's data, the only reason the setting refuses one.
  rejected: string
  onChosen: () => void
}) {
  const { t } = useLingui()
  const [error, setError] = useState<InlineError | null>(null)
  const [typing, setTyping] = useState(false)
  const apply = async (dir: string) => {
    try {
      await SetByKey(settingKey, dir, '')
    } catch (e) {
      setError(inlineError(e, rejected))
      return
    }
    setError(null)
    onChosen()
  }
  // Without a native folder dialog (server mode) the path is typed instead.
  const choose = () => {
    PickFolder(label)
      .then((dir) => (dir ? apply(dir) : undefined))
      .catch(() => setTyping(true))
  }
  return (
    <Typography variant="body2" color="text.secondary">
      {note}{' '}
      <Button size="small" onClick={choose}>
        {t`Choose folder…`}
      </Button>
      {typing ? <TypedFolder label={label} onUse={apply} /> : null}
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
