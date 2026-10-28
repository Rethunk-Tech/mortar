import { useLingui } from '@lingui/react/macro'
import { Alert, Button } from '@mui/material'
import { Lock } from 'lucide-react'
import { memo, useState } from 'react'
import { StopDialog } from '../launch/StopDialog.tsx'
import { useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { useLocked } from './useLocked.ts'

// Takes no props, so a re-render of the tab around it skips it.
export const LockedNote = memo(function LockedNoteView() {
  const { t } = useLingui()
  const locked = useLocked()
  const game = useProfiles((s) => s.game?.id ?? '')
  const [asking, setAsking] = useState(false)
  if (!locked) {
    return null
  }
  return (
    <>
      <Alert
        variant="compact"
        severity="info"
        icon={<Lock size={18} />}
        // The toolbar above already ends in its own padding.
        sx={{ mx: space.gutter, mt: 0, mb: space.gap }}
        action={
          <Button color="inherit" size="small" onClick={() => setAsking(true)}>
            {t`Stop game`}
          </Button>
        }
      >
        {t`The game is using this profile. Stop the game to change mods.`}
      </Alert>
      <StopDialog open={asking} game={game} onClose={() => setAsking(false)} />
    </>
  )
})
