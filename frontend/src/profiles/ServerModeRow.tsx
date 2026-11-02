import { useLingui } from '@lingui/react/macro'
import { MenuItem, Typography } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import {
  HasServer,
  SetServerMode,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { SelectField } from '../shell/dialog/FormDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useProfiles } from './store.ts'

// Shown only for a game whose catalog gives its profiles a dedicated server; the choice is saved as it is made.
export function ServerModeRow({
  gameId,
  profileId,
  on,
}: {
  gameId: string
  profileId: string
  on: boolean
}) {
  const { t } = useLingui()
  const replace = useProfiles((s) => s.replace)
  const { data: offered } = useLoaded(() => HasServer(gameId), [gameId], false, reportUnexpected)
  if (!offered) {
    return null
  }
  return (
    <>
      <SelectField
        label={t`Play as`}
        value={on ? 'server' : 'client'}
        onChange={(value) => {
          SetServerMode(gameId, profileId, value === 'server')
            .then((next) => next && replace(next))
            .catch(reportUnexpected)
        }}
      >
        <MenuItem value="client">{t`Client`}</MenuItem>
        <MenuItem value="server">{t`Dedicated server`}</MenuItem>
      </SelectField>
      {on ? (
        <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
          {t`Play starts the server with this profile's mods and mod settings. Its config and saves stay in this profile's own folder.`}
        </Typography>
      ) : null}
    </>
  )
}
