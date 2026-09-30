import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type {
  Diff,
  DiffSide,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  CopyMods,
  Diff as loadDiff,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { useProfiles } from './store.ts'

const paper = { paper: { sx: { bgcolor: 'rgb(40,40,48)', minWidth: 480 } } }

function sideLabel(side: DiffSide): string {
  const on = side.enabled ? '' : ' · off'
  return `${side.name} ${side.version}${on}`
}

function Section({
  title,
  rows,
  selected,
  onToggle,
}: {
  title: string
  rows: DiffSide[]
  selected: Set<string>
  onToggle: (id: string) => void
}) {
  if (rows.length === 0) {
    return null
  }
  return (
    <Box sx={{ mb: 1.5 }}>
      <Typography sx={{ fontSize: 13, fontWeight: 700, mb: 0.5, color: 'text.secondary' }}>
        {title}
      </Typography>
      {rows.map((row) => (
        <FormControlLabel
          key={row.uniqueId}
          sx={{ display: 'flex', ml: 0, mr: 0 }}
          control={
            <Checkbox
              size="small"
              checked={selected.has(row.uniqueId)}
              onChange={() => onToggle(row.uniqueId)}
            />
          }
          label={<Typography sx={{ fontSize: 14 }}>{sideLabel(row)}</Typography>}
        />
      ))}
    </Box>
  )
}

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
  const profiles = useProfiles((s) => s.profiles.filter((p) => p.id !== from?.id))
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
  const refresh = useProfiles((s) => s.refresh)
  const [diff, setDiff] = useState<Diff | null>(null)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const aId = a?.id ?? ''
  const bId = b?.id ?? ''
  const [pending, run] = usePending()
  const open = aId !== '' && bId !== ''
  useEffect(() => {
    if (!(game && aId && bId)) {
      setDiff(null)
      setSelected(new Set())
      return
    }
    loadDiff(game, aId, bId)
      .then((d) => setDiff(d))
      .catch(reportUnexpected)
  }, [game, aId, bId])
  const toggle = (id: string) =>
    setSelected((cur) => {
      const next = new Set(cur)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  const onlyA = diff?.onlyA ?? []
  const onlyB = diff?.onlyB ?? []
  const changed = diff?.changed ?? []
  const idsA = [...onlyA.map((m) => m.uniqueId), ...changed.map((p) => p.uniqueId)]
  const idsB = [...onlyB.map((m) => m.uniqueId), ...changed.map((p) => p.uniqueId)]
  const copy = (from: Profile, to: Profile, pool: string[]) => {
    const ids = pool.filter((id) => selected.has(id))
    if (ids.length === 0) {
      return
    }
    run(async () => {
      await CopyMods(game, from.id, to.id, ids)
      await refresh()
      setDiff(await loadDiff(game, a?.id ?? '', b?.id ?? ''))
    })
  }
  return (
    <Dialog
      open={open}
      onClose={pending ? undefined : onClose}
      transitionDuration={0}
      slotProps={paper}
    >
      <DialogTitle>{t`Compare ${a?.name ?? ''} and ${b?.name ?? ''}`}</DialogTitle>
      <DialogContent>
        <Section
          title={t`Only in ${a?.name ?? ''}`}
          rows={onlyA}
          selected={selected}
          onToggle={toggle}
        />
        <Section
          title={t`Only in ${b?.name ?? ''}`}
          rows={onlyB}
          selected={selected}
          onToggle={toggle}
        />
        {changed.length > 0 ? (
          <Box sx={{ mb: 1.5 }}>
            <Typography sx={{ fontSize: 13, fontWeight: 700, mb: 0.5, color: 'text.secondary' }}>
              {t`Different version or enabled state`}
            </Typography>
            {changed.map((row) => (
              <FormControlLabel
                key={row.uniqueId}
                sx={{ display: 'flex', ml: 0, mr: 0, alignItems: 'flex-start' }}
                control={
                  <Checkbox
                    size="small"
                    checked={selected.has(row.uniqueId)}
                    onChange={() => toggle(row.uniqueId)}
                  />
                }
                label={
                  <Typography sx={{ fontSize: 14 }}>
                    {t`${row.name}: ${sideLabel(row.a)} → ${sideLabel(row.b)}`}
                  </Typography>
                }
              />
            ))}
          </Box>
        ) : null}
        {onlyA.length === 0 && onlyB.length === 0 && changed.length === 0 && diff !== null ? (
          <Typography
            sx={{ color: 'text.secondary' }}
          >{t`These profiles have the same mods.`}</Typography>
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button
          disabled={pending || !idsA.some((id) => selected.has(id))}
          onClick={() => a && b && copy(a, b, idsA)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Copy selected to ${b?.name ?? ''}`}
        </Button>
        <Button
          disabled={pending || !idsB.some((id) => selected.has(id))}
          onClick={() => a && b && copy(b, a, idsB)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Copy selected to ${a?.name ?? ''}`}
        </Button>
        <Button onClick={onClose} disabled={pending}>
          {t`Close`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
