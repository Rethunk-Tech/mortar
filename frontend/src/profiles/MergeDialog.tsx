import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  FormLabel,
  MenuItem,
  Radio,
  RadioGroup,
  TextField,
  Typography,
} from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useEffect, useId, useState } from 'react'
import type {
  MergePreview as Merged,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  MergeInto,
  MergePreview as PreviewMerge,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useOrderedProfiles } from '../game/useSidebarProfiles.ts'
import { lockedIn, useLaunchLocks } from '../mods/useLocked.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { useProfiles } from './store.ts'

// The binding types a Go slice as nullable.
type Preview = { [K in keyof Merged]-?: NonNullable<Merged[K]> }

function AddsLine({ adds }: { adds: Preview['adds'] }) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <Typography>{plural(adds.length, { one: 'Adds # mod', other: 'Adds # mods' })}</Typography>
        {adds.length > 0 ? (
          <Button
            size="small"
            aria-expanded={open}
            onClick={() => setOpen(!open)}
            startIcon={open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          >
            {open ? t`Hide mods` : t`Show mods`}
          </Button>
        ) : null}
      </Box>
      {open ? (
        <Box component="ul" sx={{ m: 0, p: 0, pl: 2.5, maxHeight: 160, overflowY: 'auto' }}>
          {adds.map((side) => (
            <Typography component="li" key={side.id} sx={{ fontSize: 13 }}>
              {side.name}
            </Typography>
          ))}
        </Box>
      ) : null}
    </Box>
  )
}

function Summary({
  preview,
  target,
  newerWins,
  onNewerWins,
}: {
  preview: Preview
  target: Profile
  newerWins: boolean
  onNewerWins: (value: boolean) => void
}) {
  const { t } = useLingui()
  const both = preview.both.length
  const updates = preview.both.filter((b) => b.sourceNewer).length
  const targetName = target.name
  const labelId = useId()
  // Nothing to add and no newer version in the source: say so instead of counting zeros, so the disabled Merge
  // button has its reason in view.
  if (preview.adds.length === 0 && updates === 0) {
    return (
      <Typography color="text.secondary">{t`Nothing to merge: ${targetName} already has every mod at the same or a newer version.`}</Typography>
    )
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
      {preview.adds.length > 0 ? <AddsLine adds={preview.adds} /> : null}
      {both > 0 ? (
        <Typography>
          {plural(both, { one: '# mod is in both', other: '# mods are in both' })}
        </Typography>
      ) : null}
      {preview.targetOnly > 0 ? (
        <Typography>
          {plural(preview.targetOnly, {
            one: `Keeps # mod only ${targetName} has`,
            other: `Keeps # mods only ${targetName} has`,
          })}
        </Typography>
      ) : null}
      {updates > 0 ? (
        <FormControl>
          <FormLabel id={labelId}>{t`For mods in both`}</FormLabel>
          <RadioGroup
            aria-labelledby={labelId}
            value={newerWins ? 'newer' : 'keep'}
            onChange={(e) => onNewerWins(e.target.value === 'newer')}
          >
            <FormControlLabel
              value="keep"
              control={<Radio size="small" />}
              label={t`Keep ${targetName}'s versions`}
            />
            <FormControlLabel
              value="newer"
              control={<Radio size="small" />}
              label={plural(updates, {
                one: 'Use the newer version (# mod updates)',
                other: 'Use the newer version (# mods update)',
              })}
            />
          </RadioGroup>
        </FormControl>
      ) : null}
    </Box>
  )
}

function MergeBody({ source, onClose }: { source: Profile; onClose: () => void }) {
  const { t } = useLingui()
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const refresh = useProfiles((s) => s.refresh)
  const openProfile = useProfiles((s) => s.open)
  const { profiles } = useOrderedProfiles(gameId)
  const others = profiles.filter((p) => p.id !== source.id)
  const [targetId, setTargetId] = useState(others[0]?.id ?? '')
  const [loaded, setLoaded] = useState<{ id: string; preview: Preview } | null>(null)
  const [newerWins, setNewerWins] = useState(false)
  const [pending, run] = usePending()
  const launch = useLaunchLocks()
  const target = others.find((p) => p.id === targetId)
  const preview = loaded?.id === targetId ? loaded.preview : null
  useEffect(() => {
    if (!targetId) {
      return
    }
    let stale = false
    PreviewMerge(gameId, source.id, targetId)
      .then((p) => {
        if (!stale) {
          setLoaded({
            id: targetId,
            preview: { adds: p.adds ?? [], both: p.both ?? [], targetOnly: p.targetOnly },
          })
        }
      })
      .catch(reportUnexpected)
    return () => {
      stale = true
    }
  }, [gameId, source.id, targetId])
  const updates = preview ? preview.both.filter((b) => b.sourceNewer).length : 0
  const changes = preview ? preview.adds.length + (newerWins ? updates : 0) : 0
  const locked = lockedIn(launch, targetId)
  const reason = locked ? t`Stop the game to change mods.` : ''
  const merge = () => {
    if (!target) {
      return
    }
    const targetName = target.name
    run(
      async () => {
        await MergeInto(gameId, source.id, target.id, newerWins)
        await refresh()
        onClose()
        openProfile(target.id)
        useToasts.getState().push({
          kind: 'success',
          title: plural(changes, {
            one: `Merged # mod into ${targetName}`,
            other: `Merged # mods into ${targetName}`,
          }),
        })
      },
      { errorTitle: t`Could not merge into ${targetName}` },
    )
  }
  return (
    <>
      <DialogTitle>{t`Merge ${source.name} into…`}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: space.pad }}>
        <TextField
          select={true}
          size="small"
          label={t`Target profile`}
          value={targetId}
          onChange={(e) => setTargetId(e.target.value)}
          disabled={pending}
          sx={{ mt: 1 }}
        >
          {others.map((p) => (
            <MenuItem key={p.id} value={p.id}>
              {p.name}
            </MenuItem>
          ))}
        </TextField>
        {preview && target ? (
          <Summary
            preview={preview}
            target={target}
            newerWins={newerWins}
            onNewerWins={setNewerWins}
          />
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={pending}>
          {t`Cancel`}
        </Button>
        <DisabledReason title={reason} disabled={reason !== ''}>
          <Button
            variant="contained"
            disabled={pending || !preview || changes === 0 || locked}
            onClick={merge}
          >
            {t`Merge`}
          </Button>
        </DisabledReason>
      </DialogActions>
    </>
  )
}

export function MergeDialog({ source, onClose }: { source: Profile | null; onClose: () => void }) {
  return (
    <Dialog
      open={source !== null}
      onClose={onClose}
      slotProps={{ paper: { sx: { width: 460, maxWidth: 'calc(100vw - 64px)' } } }}
    >
      {source ? <MergeBody source={source} onClose={onClose} /> : null}
    </Dialog>
  )
}
