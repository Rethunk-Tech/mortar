import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { Mod } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../../profiles/store.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { useTypedConfig } from './store.ts'

// Opens the typed editor on the mod.
export function EditConfigButton({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  return (
    <Button
      size="small"
      variant="outlined"
      onClick={() => {
        const { game, openId } = useProfiles.getState()
        if (game && openId) {
          useTypedConfig
            .getState()
            .open(mod, { game: game.id, profile: openId, key: mod.key, id: mod.id })
            .catch(reportUnexpected)
        }
      }}
    >
      {t`Edit config`}
    </Button>
  )
}
