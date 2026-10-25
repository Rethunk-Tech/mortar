import { useLingui } from '@lingui/react/macro'
import { MenuItem } from '@mui/material'
import { SetLoader } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useGameInfo } from '../games/info.ts'
import { SelectField } from '../shell/dialog/FormDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useProfiles } from './store.ts'

// Shown only for a game whose catalog lists more than one loader; the choice is saved as it is made.
export function LoaderPicker({
  gameId,
  profileId,
  loader,
}: {
  gameId: string
  profileId: string
  loader: string
}) {
  const { t } = useLingui()
  const replace = useProfiles((s) => s.replace)
  const loaders = useGameInfo(gameId)?.loaders ?? []
  if (loaders.length < 2) {
    return null
  }
  return (
    <SelectField
      label={t`Mod loader`}
      value={loader || loaders[0]?.id || ''}
      onChange={(value) => {
        SetLoader(gameId, profileId, value)
          .then((next) => next && replace(next))
          .catch(reportUnexpected)
      }}
    >
      {loaders.map((l) => (
        <MenuItem key={l.id} value={l.id}>
          {l.name}
        </MenuItem>
      ))}
    </SelectField>
  )
}
