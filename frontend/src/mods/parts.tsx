import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Switch,
  Tooltip,
} from '@mui/material'
import { ArrowUp, Ban, Pin, TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { dependentsOf } from './dependents.ts'
import { useDescribe } from './describe.ts'
import { LockedReason } from './LockedReason.tsx'
import { lastRunSummary, useLastRun } from './lastRun.ts'
import { concerns, entryOf, modId, nexusIdOf, problemsOf, siblingsOf, updateFor } from './lookup.ts'
import { NewDot } from './NewSince.tsx'
import { useNexusEntry } from './nexusDetails.ts'
import { goneCaption, nexusPageMark, offersNexusDownload } from './nexusMark.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'
import { useLocked } from './useLocked.ts'

const HASH_MULTIPLIER = 31
const UINT32_BITS = 32
const UINT32_RANGE = 2 ** UINT32_BITS
const HUE_DEGREES = 360
const TILE_FONT_RATIO = 0.5
const DEFAULT_TILE_SIZE = 48
const SMALL_TILE = 32

function hash(s: string): number {
  let h = 0
  for (const c of s) {
    h = (h * HASH_MULTIPLIER + (c.codePointAt(0) ?? 0)) % UINT32_RANGE
  }
  return h
}

// The mod's Nexus picture when it has one that loads, else its first letter on a colour from its UniqueID.
export function LetterTile({
  mod,
  size = DEFAULT_TILE_SIZE,
  fresh = false,
}: {
  mod: Pick<Mod, 'uniqueId' | 'name'> & { picture?: string }
  size?: number
  fresh?: boolean
}) {
  const [failed, setFailed] = useState<string | null>(null)
  const picture = mod.picture && failed !== mod.picture ? mod.picture : ''
  return (
    <Box sx={{ position: 'relative', width: size, height: size, flexShrink: 0 }}>
      <Box
        aria-hidden={true}
        className="tile"
        sx={{
          width: size,
          height: size,
          borderRadius: size < SMALL_TILE ? '4px' : '6px',
          display: 'grid',
          placeItems: 'center',
          overflow: 'hidden',
          fontWeight: 700,
          fontSize: size * TILE_FONT_RATIO,
          bgcolor: `hsl(${hash(mod.uniqueId.toLowerCase()) % HUE_DEGREES} 35% 38% / 0.85)`,
        }}
      >
        {picture ? (
          <Box
            component="img"
            alt=""
            src={`/mod-picture/?u=${encodeURIComponent(picture)}`}
            onError={() => setFailed(picture)}
            sx={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        ) : (
          (Array.from(mod.name)[0] ?? '?').toUpperCase()
        )}
      </Box>
      <NewDot show={fresh} />
    </Box>
  )
}

export function ProblemBadge({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const describe = useDescribe()
  const result = useMods((s) => s.problems)
  const mine = problemsOf(result).filter((p) => concerns(p, mod))
  if (mine.length === 0) {
    return null
  }
  const text = mine.map(describe).join(' ')
  return (
    <Tooltip title={text}>
      <Box
        role="img"
        aria-label={t`Problem: ${text}`}
        sx={{ display: 'flex', flexShrink: 0, color: 'warning.main' }}
      >
        <TriangleAlert size={16} />
      </Box>
    </Tooltip>
  )
}

export function NexusGoneBadge({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const profile = useProfiles(openProfileOf)
  const nexusId = profile ? nexusIdOf(profile, mod) : 0
  const page = useNexusEntry(nexusId)?.details?.page
  const mark = nexusPageMark(page?.status, page?.available, page?.updated, page?.created)
  if (mark.kind === '') {
    return null
  }
  const text = goneCaption(mark, {
    hidden: t`Hidden on Nexus`,
    hiddenDated: t`Hidden on Nexus · ${mark.date}`,
    removed: t`Removed from Nexus`,
    removedDated: t`Removed from Nexus · ${mark.date}`,
  })
  return (
    <Tooltip title={text}>
      <Box
        role="img"
        aria-label={text}
        sx={{ display: 'flex', flexShrink: 0, color: 'warning.main' }}
      >
        <Ban size={16} />
      </Box>
    </Tooltip>
  )
}

export function LastRunBadge({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const hit = useLastRun((s) => s.byId[mod.uniqueId])
  if (!hit || (hit.errors === 0 && hit.warnings === 0)) {
    return null
  }
  const errors = hit.errors ? t`${plural(hit.errors, { one: '# error', other: '# errors' })}` : ''
  const warnings = hit.warnings
    ? t`${plural(hit.warnings, { one: '# warning', other: '# warnings' })}`
    : ''
  const text = lastRunSummary(hit)
  const chipSx = {
    minWidth: 16,
    height: 16,
    px: 0.4,
    borderRadius: '4px',
    color: 'common.white',
    fontSize: 11,
    fontWeight: 700,
    lineHeight: '16px',
    textAlign: 'center',
    flexShrink: 0,
  } as const
  return (
    <Box sx={{ display: 'flex', flexShrink: 0, gap: 0.4 }} aria-label={t`Last run: ${text}`}>
      {hit.errors > 0 ? (
        <Box role="img" aria-label={errors} sx={{ ...chipSx, bgcolor: 'error.main' }}>
          {hit.errors}
        </Box>
      ) : null}
      {hit.warnings > 0 ? (
        <Box role="img" aria-label={warnings} sx={{ ...chipSx, bgcolor: 'warning.main' }}>
          {hit.warnings}
        </Box>
      ) : null}
    </Box>
  )
}

export function PinBadge({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const profile = useProfiles(openProfileOf)
  const entry = entryOf(profile, mod.key)
  if (!entry?.pinned) {
    return null
  }
  const text = entry.pinReason ? t`Pinned: ${entry.pinReason}` : t`Pinned at this version`
  return (
    <Tooltip title={text}>
      <Box
        role="img"
        aria-label={text}
        sx={{ display: 'flex', flexShrink: 0, color: 'text.secondary' }}
      >
        <Pin size={16} />
      </Box>
    </Tooltip>
  )
}

export function UpdateBadge({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const profile = useProfiles(openProfileOf)
  const update = useUpdates((s) => updateFor(s.updates, mod, profile))
  const nexusId = profile ? nexusIdOf(profile, mod) : 0
  const page = useNexusEntry(nexusId)?.details?.page
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  if (!update) {
    return null
  }
  if (update.nexusId > 0 && !offersNexusDownload(page?.status, page?.available)) {
    return null
  }
  const text = update.unofficial
    ? t`Unofficial update available: ${update.version}`
    : t`Update available: ${update.installed} → ${update.version}`
  return (
    <>
      <Tooltip title={text}>
        <IconButton
          aria-label={text}
          aria-haspopup="menu"
          aria-expanded={anchor !== null}
          onClick={(e) => {
            e.stopPropagation()
            setAnchor(e.currentTarget)
          }}
          sx={{ p: 0, flexShrink: 0, color: 'primary.main', borderRadius: '4px' }}
        >
          <ArrowUp size={16} />
        </IconButton>
      </Tooltip>
      <Menu
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        onClick={(e) => e.stopPropagation()}
      >
        <MenuItem
          onClick={() => {
            setAnchor(null)
            setSkipVersion(mod, update.version).catch(reportUnexpected)
          }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <Ban size={16} />
          </ListItemIcon>
          <ListItemText>{t`Skip this update`}</ListItemText>
        </MenuItem>
      </Menu>
    </>
  )
}

export function ModSwitch({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const setEnabled = useMods((s) => s.setEnabled)
  const locked = useLocked()
  return (
    <LockedReason locked={locked}>
      <Switch
        size="small"
        checked={mod.enabled}
        disabled={locked}
        onChange={(e) => {
          setEnabled(mod, e.target.checked).catch(reportUnexpected)
        }}
        onClick={(e) => e.stopPropagation()}
        slotProps={{
          input: {
            'aria-label': mod.enabled ? t`Switch off ${mod.name}` : t`Switch on ${mod.name}`,
          },
        }}
      />
    </LockedReason>
  )
}

export function ShowFilesButton({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const showFiles = useMods((s) => s.showFiles)
  return (
    <Button
      variant="outlined"
      onClick={() => {
        showFiles(mod).catch(reportUnexpected)
      }}
    >
      {t`Show files`}
    </Button>
  )
}

export function RemoveButton({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const askRemove = useMods((s) => s.askRemove)
  const locked = useLocked()
  return (
    <LockedReason locked={locked}>
      <Button variant="outlined" color="error" disabled={locked} onClick={() => askRemove(mod)}>
        {t`Remove`}
      </Button>
    </LockedReason>
  )
}

export function RemoveDialog() {
  const { t } = useLingui()
  const removing = useMods((s) => s.removing)
  const mods = useMods((s) => s.mods)
  const askRemove = useMods((s) => s.askRemove)
  const removeMany = useMods((s) => s.removeMany)
  const locked = useLocked()
  const one = removing.length === 1 ? removing[0] : null
  const extra = [
    ...new Set(
      removing.flatMap((m) =>
        siblingsOf(mods, m)
          .filter((s) => !removing.some((x) => modId(x) === modId(s)))
          .map((s) => s.name),
      ),
    ),
  ]
  const dependents = dependentsOf(mods, removing)
  const close = () => askRemove(null)
  let body = t`Their folders in this profile are deleted. You can undo this.`
  if (extra.length > 0) {
    body = t`Mods from the same download are removed together: ${extra.join(', ')}. You can undo this.`
  } else if (one) {
    body = t`Its folder in this profile is deleted. You can undo this.`
  }
  const drop = (list: typeof removing) => {
    close()
    if (list.length > 0) {
      removeMany(list).catch(reportUnexpected)
    }
  }
  const needLine =
    dependents.length > 0
      ? t`${plural(dependents.length, { one: '# mod needs this', other: '# mods need this' })}: ${dependents.map((m) => m.name).join(', ')}`
      : ''
  const allCount = removing.length + dependents.length
  return (
    <Dialog open={removing.length > 0} onClose={close}>
      <DialogTitle>
        {one
          ? t`Remove ${one.name} from this profile?`
          : t`${plural(removing.length, { one: 'Remove # mod from this profile?', other: 'Remove # mods from this profile?' })}`}
      </DialogTitle>
      <DialogContent>
        <DialogContentText>{body}</DialogContentText>
        {needLine ? <DialogContentText sx={{ mt: 1 }}>{needLine}</DialogContentText> : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={close}>{t`Cancel`}</Button>
        {dependents.length > 0 ? (
          <>
            <LockedReason locked={locked}>
              <Button color="error" disabled={locked} onClick={() => drop(removing)}>
                {t`Remove anyway`}
              </Button>
            </LockedReason>
            <LockedReason locked={locked}>
              <Button
                variant="contained"
                color="error"
                disabled={locked}
                onClick={() => drop([...removing, ...dependents])}
              >
                {t`${plural(allCount, { one: 'Remove all # mod', other: 'Remove all # mods' })}`}
              </Button>
            </LockedReason>
          </>
        ) : (
          <LockedReason locked={locked}>
            <Button
              variant="contained"
              color="error"
              disabled={locked}
              onClick={() => drop(removing)}
            >
              {t`Remove`}
            </Button>
          </LockedReason>
        )}
      </DialogActions>
    </Dialog>
  )
}
