import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { useMemo } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { CopyMods } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { usePending } from '../toasts/usePending.ts'
import { CompareBody } from './CompareBody.tsx'
import { compareProfiles } from './compare.ts'
import { useProfiles } from './store.ts'

const paper = { paper: { sx: { minWidth: 520, maxWidth: 720 } } }

export function PickCompareDialog({
  from,
  onPicked,
  onClose,
}: {
  from: Profile | null
  onPicked: (other: Profile) => void
  onClose: () => void
}) {
  const { t } = useLingui()
  const all = useProfiles((s) => s.profiles)
  const profiles = all.filter((p) => p.id !== from?.id)
  return (
    <Dialog open={from !== null} onClose={onClose} transitionDuration={0} slotProps={paper}>
      <DialogTitle>{t`Compare ${from?.name ?? ''} with…`}</DialogTitle>
      <DialogContent>
        {profiles.length === 0 ? (
          <Typography
            sx={{ color: 'text.secondary' }}
          >{t`No other profile of this game.`}</Typography>
        ) : (
          profiles.map((p) => (
            <Button
              key={p.id}
              color="inherit"
              onClick={() => onPicked(p)}
              sx={{
                display: 'block',
                width: 1,
                justifyContent: 'flex-start',
                whiteSpace: 'nowrap',
              }}
            >
              {p.name}
            </Button>
          ))
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
      </DialogActions>
    </Dialog>
  )
}

export function CompareDialog({
  a,
  b,
  onClose,
}: {
  a: Profile | null
  b: Profile | null
  onClose: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const listed = useProfiles((s) => s.profiles)
  const refresh = useProfiles((s) => s.refresh)
  const aId = a?.id ?? ''
  const bId = b?.id ?? ''
  const open = aId !== '' && bId !== ''
  const profileA = listed.find((p) => p.id === aId) ?? a
  const profileB = listed.find((p) => p.id === bId) ?? b
  const diff = useMemo(() => {
    if (!(profileA && profileB)) {
      return null
    }
    return compareProfiles(profileA, profileB)
  }, [profileA, profileB])
  const [pending, run] = usePending()

  const copyOne = (from: Profile, to: Profile, uniqueId: string) => {
    run(async () => {
      await CopyMods(game, from.id, to.id, [uniqueId])
      await refresh()
    })
  }

  const aName = profileA?.name ?? ''
  const bName = profileB?.name ?? ''

  return (
    <Dialog
      open={open}
      onClose={pending ? undefined : onClose}
      transitionDuration={0}
      slotProps={paper}
    >
      <DialogTitle>{t`Compare ${aName} and ${bName}`}</DialogTitle>
      <DialogContent>
        {diff && profileA && profileB ? (
          <CompareBody
            diff={diff}
            profileA={profileA}
            profileB={profileB}
            aName={aName}
            bName={bName}
            pending={pending}
            onCopy={copyOne}
          />
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={pending}>{t`Close`}</Button>
      </DialogActions>
    </Dialog>
  )
}
