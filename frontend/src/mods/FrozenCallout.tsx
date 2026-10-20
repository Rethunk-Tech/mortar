import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { memo } from 'react'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { setFrozen } from './freeze.ts'
import { isFrozen } from './frozen.ts'
import { ListCallout } from './ListCallout.tsx'

// Shown above the mod list while the profile is frozen, with the way out.
export const FrozenCallout = memo(function FrozenCalloutView() {
  const { t } = useLingui()
  const frozen = useProfiles((s) => isFrozen(openProfileOf(s)))
  if (!frozen) {
    return null
  }
  return (
    <ListCallout
      text={t`This profile is frozen: its mods stay on their versions and no updates are offered.`}
      actions={
        <Button size="small" variant="text" onClick={() => setFrozen(false)}>
          {t`Unfreeze`}
        </Button>
      }
    />
  )
})
