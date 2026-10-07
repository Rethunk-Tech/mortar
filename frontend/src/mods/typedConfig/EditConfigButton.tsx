import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { Mod } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { openConfigPage } from './configList.ts'

// Opens the Config page with the mod selected.
export function EditConfigButton({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  return (
    <Button size="small" variant="outlined" onClick={() => openConfigPage(mod.id)}>
      {t`Edit config`}
    </Button>
  )
}
