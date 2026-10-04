import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  ButtonBase,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  LinearProgress,
  Switch,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type {
  AssetMapPage,
  AssetTarget,
  AssetTouch,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { AssetMap } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { sameId } from './lookup.ts'
import { LinkedText } from './ModLinks.tsx'
import { applyWins } from './problemFix/applyWins.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const filterDelayMs = 200
const chipSx = { height: 22, borderRadius: '4px' } as const
const buttonSx = { height: 24, flexShrink: 0, whiteSpace: 'nowrap' } as const

function TouchActions({
  target,
  mod,
  onChanged,
}: {
  target: AssetTarget
  mod: AssetTouch
  onChanged: () => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  if (mod.winner) {
    return null
  }
  const entry = mods.find((m) => m.key === mod.modKey && sameId(m.uniqueId, mod.modId))
  const others = (target.mods ?? [])
    .map((m) => m.modId)
    .filter((id, i, ids) => !sameId(id, mod.modId) && ids.findIndex((x) => sameId(x, id)) === i)
  return (
    <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
      <Box sx={{ display: 'flex', gap: 0.5, ml: 'auto' }}>
        {mod.canWin ? (
          <Button
            size="small"
            variant="outlined"
            color="inherit"
            disabled={locked}
            title={t`Load ${mod.modName} after the other mods here, so its edit applies last`}
            onClick={() => {
              applyWins(mod.modKey, others, -1, true).then(onChanged).catch(reportUnexpected)
            }}
            sx={buttonSx}
          >
            {t`Make win`}
          </Button>
        ) : null}
        {entry ? (
          <Button
            size="small"
            color="inherit"
            variant="text"
            disabled={locked}
            onClick={() => {
              setEnabled(entry, false).then(onChanged).catch(reportUnexpected)
            }}
            sx={buttonSx}
          >
            {t`Switch off`}
          </Button>
        ) : null}
      </Box>
    </DisabledReason>
  )
}

function ModTouches({ target, onChanged }: { target: AssetTarget; onChanged: () => void }) {
  const { t } = useLingui()
  const many = new Set((target.mods ?? []).map((m) => m.modId.toLowerCase())).size > 1
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, mt: 0.75 }}>
      {(target.mods ?? []).map((mod) => {
        const name = mod.modName || mod.modId
        return (
          <Box
            key={`${mod.modId}-${mod.action}-${String(mod.index)}-${mod.source ?? ''}-${mod.dataKey ?? ''}`}
            sx={{ display: 'flex', alignItems: 'center', gap: 0.75, flexWrap: 'wrap' }}
          >
            <Typography sx={{ fontSize: 13, fontWeight: mod.winner ? 'bold' : 'normal' }}>
              <LinkedText text={name} links={[{ name, key: mod.modKey, uniqueId: mod.modId }]} />
            </Typography>
            {mod.action ? <Chip size="small" label={mod.action} sx={chipSx} /> : null}
            {mod.priority ? (
              <Chip size="small" variant="outlined" label={mod.priority} sx={chipSx} />
            ) : null}
            {mod.winner ? (
              <Chip size="small" color="success" label={t`winner`} sx={chipSx} />
            ) : null}
            {many ? <TouchActions target={target} mod={mod} onChanged={onChanged} /> : null}
          </Box>
        )
      })}
    </Box>
  )
}

function TargetRow({
  target,
  expanded,
  onToggle,
  onChanged,
}: {
  target: AssetTarget
  expanded: boolean
  onToggle: () => void
  onChanged: () => void
}) {
  const { t } = useLingui()
  const count = new Set((target.mods ?? []).map((m) => m.modId.toLowerCase())).size
  const many = count > 1
  const label = target.key ? `${target.target} ${target.key}` : target.target
  return (
    <Box sx={{ bgcolor: 'background.paper', borderRadius: 1, px: 1.25, py: 1 }}>
      <ButtonBase
        aria-expanded={expanded}
        onClick={onToggle}
        sx={{
          display: 'flex',
          alignItems: 'baseline',
          justifyContent: 'flex-start',
          gap: 1,
          width: '100%',
          textAlign: 'left',
        }}
      >
        <Typography component="span" sx={{ fontSize: 14, fontWeight: 600, flex: 1 }}>
          {label}
        </Typography>
        <Typography
          component="span"
          title={
            many ? t`More than one mod changes this; the winner's version is used.` : undefined
          }
          sx={{
            fontSize: 13,
            color: many ? 'warning.main' : 'text.secondary',
            whiteSpace: 'nowrap',
          }}
        >
          {plural(count, { one: 'Changed by # mod', other: 'Changed by # mods' })}
        </Typography>
      </ButtonBase>
      {expanded ? <ModTouches target={target} onChanged={onChanged} /> : null}
    </Box>
  )
}

function useAssetPages(open: boolean, filter: string, shared: boolean) {
  const game = useProfiles((s) => s.game)
  const openId = useProfiles((s) => s.openId)
  const [page, setPage] = useState<AssetMapPage | null>(null)
  useEffect(() => {
    if (!(open && game && openId)) {
      return
    }
    const handle = globalThis.setTimeout(() => {
      AssetMap(game.id, openId, filter.trim(), shared, 0).then(setPage).catch(reportUnexpected)
    }, filterDelayMs)
    return () => {
      globalThis.clearTimeout(handle)
    }
  }, [open, game, openId, filter, shared])
  const fetchFrom = (offset: number, before: AssetTarget[]) => {
    if (!(game && openId)) {
      return
    }
    AssetMap(game.id, openId, filter.trim(), shared, offset)
      .then((next) => {
        setPage({ ...next, targets: [...before, ...(next.targets ?? [])] })
      })
      .catch(reportUnexpected)
  }
  const targets = page?.targets ?? []
  return {
    page,
    more: () => fetchFrom(targets.length, targets),
    reload: () => fetchFrom(0, []),
  }
}

export function AssetMapDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const [filter, setFilter] = useState('')
  const [shared, setShared] = useState(true)
  const [expanded, setExpanded] = useState<string | null>(null)
  const { page, more, reload } = useAssetPages(open, filter, shared)
  const targets = page?.targets ?? []
  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth={true} scroll="paper">
      <DialogTitle>{t`Asset map`}</DialogTitle>
      <DialogContent dividers={true} sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
        <Typography sx={{ fontSize: 14, color: 'text.secondary' }}>
          {t`Every game asset the profile's mods change, and which mod wins it. Open one to see each mod's edit.`}
        </Typography>
        <SearchField
          label={t`Find an asset`}
          placeholder={t`Find an asset, for example Maps/Town`}
          value={filter}
          onChange={setFilter}
        />
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
          <FormControlLabel
            control={<Switch size="small" checked={shared} onChange={(_, on) => setShared(on)} />}
            label={t`Only assets more than one mod changes`}
          />
          {page ? (
            <Typography sx={{ fontSize: 13, color: 'text.secondary', ml: 'auto' }}>
              {t`${plural(page.all, { one: '# asset', other: '# assets' })}, ${page.shared} changed by more than one mod`}
            </Typography>
          ) : null}
        </Box>
        {page === null ? (
          <LinearProgress />
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
            {targets.map((target) => {
              const id = `${target.target}\t${target.key ?? ''}`
              return (
                <TargetRow
                  key={id}
                  target={target}
                  expanded={expanded === id}
                  onToggle={() => setExpanded((cur) => (cur === id ? null : id))}
                  onChanged={reload}
                />
              )
            })}
            {targets.length < page.total ? (
              <Button color="inherit" onClick={more} sx={{ alignSelf: 'center' }}>
                {t`Show more`}
              </Button>
            ) : null}
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Close`}</Button>
      </DialogActions>
    </Dialog>
  )
}
