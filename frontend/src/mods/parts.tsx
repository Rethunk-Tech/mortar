import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControlLabel,
  IconButton,
  Menu,
  Switch,
} from '@mui/material'
import { ArrowUp, Ban, CircleCheck, CircleX, Pin, TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { hasSourceLogo, pictureMarkSx } from '../brand/sources/sourceIcons.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { errorsLabel } from '../i18n/counts.ts'
import { listNames } from '../i18n/list.ts'
import { useLive } from '../launch/live.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { destructiveSx } from '../shell/destructive.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { Tooltip } from '../shell/Tooltip.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { dependentsOf, idKey, localId } from './dependents.ts'
import { useDescribe } from './describe.ts'
import { unneededImplicit } from './disableUnneeded.ts'
import { FavouriteStar } from './FavouriteStar.tsx'
import { entryFavouriteKey } from './favourites.ts'
import { readsImplicit } from './installReason.ts'
import { LockedReason } from './LockedReason.tsx'
import { lastRunSummary, useLastRun } from './lastRun.ts'
import {
  concerns,
  entryOf,
  modId,
  nexusIdOf,
  problemsOf,
  siblingsOf,
  sourceKind,
  updateFor,
} from './lookup.ts'
import { NewDot } from './NewSince.tsx'
import { useNexusPage } from './nexusDetails.ts'
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
const TILE_MARK_PX = 12
const SMALL_TILE = 32

function hash(s: string): number {
  let h = 0
  for (const c of s) {
    h = (h * HASH_MULTIPLIER + (c.codePointAt(0) ?? 0)) % UINT32_RANGE
  }
  return h
}

const MAX_NAMED = 12

// The mod's Nexus picture when it has one that loads, else its first letter on a colour from its UniqueID.
function RemoveDialogBody({ removing }: { removing: Mod[] }) {
  const { t } = useLingui()
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
  const profile = useProfiles(openProfileOf)
  const removingKeys = new Set(removing.map((m) => m.key))
  const optional = (profile?.entries ?? []).filter((e) =>
    removingKeys.has(e.overlayOf ?? ''),
  ).length
  // Libraries only the removed mods needed, offered for removal with them (apt's autoremove).
  const unneeded = unneededImplicit(
    mods,
    mods.filter((m) => removingKeys.has(m.key)),
    (m) => readsImplicit(entryOf(profile, m.key)?.implicit, m, mods),
    true,
  )
  const [withUnneeded, setWithUnneeded] = useState(true)
  const close = () => askRemove(null)
  let body = t`Their folders are removed from this profile. You can undo this.`
  if (extra.length > 0) {
    body = t`Mods from the same download are removed together: ${listNames(extra)}. You can undo this.`
  } else if (one) {
    body = t`Its folder is removed from this profile. You can undo this.`
  }
  const drop = (chosen: typeof removing) => {
    close()
    const list = withUnneeded ? [...chosen, ...unneeded] : chosen
    if (list.length > 0) {
      removeMany(list).catch(reportUnexpected)
    }
  }
  const needLine =
    dependents.length > 0
      ? t`${plural(dependents.length, { one: '# mod needs this', other: '# mods need this' })}: ${listNames(dependents.map((m) => m.name))}`
      : ''
  const allCount = removing.length + dependents.length
  return (
    <>
      <DialogTitle>
        {one
          ? t`Remove ${one.name} from this profile?`
          : t`${plural(removing.length, { one: 'Remove # mod from this profile?', other: 'Remove # mods from this profile?' })}`}
      </DialogTitle>
      <DialogContent>
        <DialogContentText>{body}</DialogContentText>
        {removing.length > 1 ? (
          <DialogContentText sx={{ mt: 1 }}>
            {listNames(
              removing.map((m) => m.name),
              MAX_NAMED,
            )}
          </DialogContentText>
        ) : null}
        {optional > 0 ? (
          <DialogContentText sx={{ mt: 1 }}>
            {t`${plural(optional, { one: 'Also removes # optional file that goes on top of it.', other: 'Also removes # optional files that go on top of it.' })}`}
          </DialogContentText>
        ) : null}
        {needLine ? <DialogContentText sx={{ mt: 1 }}>{needLine}</DialogContentText> : null}
        {unneeded.length > 0 ? (
          <FormControlLabel
            sx={{ mt: 1 }}
            control={
              <Checkbox
                checked={withUnneeded}
                onChange={(e) => setWithUnneeded(e.target.checked)}
              />
            }
            label={t`${plural(unneeded.length, { one: 'Also remove # library only this needed', other: 'Also remove # libraries only these needed' })}: ${listNames(unneeded.map((m) => m.name))}`}
          />
        ) : null}
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
    </>
  )
}

// The panel footer's two buttons share one height.
const footerButtonSx = { height: space.control } as const

// source and starKey lay the mod's source mark and its favourite star over the picture's corners.
export function LetterTile({
  mod,
  size = DEFAULT_TILE_SIZE,
  fresh = false,
  source = '',
  starKey = '',
}: {
  mod: Pick<Mod, 'id' | 'name'> & { picture?: string }
  size?: number
  fresh?: boolean
  source?: string
  starKey?: string
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
          bgcolor: `hsl(${hash(localId(mod.id).toLowerCase()) % HUE_DEGREES} 35% 38% / 0.85)`,
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
      {hasSourceLogo(source) ? (
        <Box
          role="img"
          aria-label={sourceLabel(source)}
          title={sourceLabel(source)}
          sx={{ ...pictureMarkSx, position: 'absolute', top: 2, right: 2 }}
        >
          <SourceLogo id={source} size={TILE_MARK_PX} />
        </Box>
      ) : null}
      {starKey ? <FavouriteStar starKey={starKey} onPicture={true} /> : null}
    </Box>
  )
}

// A profile's mod with its source and favourite star on the picture.
export function ProfileModTile({
  mod,
  profile,
  fresh,
  size,
}: {
  mod: Mod
  profile: Profile
  fresh: boolean
  size: number
}) {
  return (
    <LetterTile
      mod={mod}
      size={size}
      fresh={fresh}
      source={sourceKind(profile, mod)}
      starKey={entryFavouriteKey(entryOf(profile, mod.key))}
    />
  )
}

// Most cards have no problem, and describing one reads every mod's name, so only a card with a problem subscribes to
// the mods list.
export function ProblemBadge({ mod }: { mod: Mod }) {
  const has = useMods((s) => problemsOf(s.problems).some((p) => concerns(p, mod)))
  return has ? <ProblemMark mod={mod} /> : null
}

function ProblemMark({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const describe = useDescribe()
  const result = useMods((s) => s.problems)
  const text = problemsOf(result)
    .filter((p) => concerns(p, mod))
    .map(describe)
    .join(' ')
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

// LiveBadge says, while the profile's game runs, whether any plugin of a package that ships plugins has loaded.
export function LiveBadge({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const loaded = useLive((s) => (s.profile === openId ? s.loaded[idKey(mod.id)] : undefined))
  if (loaded === undefined) {
    return null
  }
  const text = loaded ? t`Loaded in the running game` : t`Not loaded in the running game`
  return (
    <Tooltip title={text}>
      <Box
        role="img"
        aria-label={text}
        sx={{ display: 'flex', flexShrink: 0, color: loaded ? 'success.main' : 'error.main' }}
      >
        {loaded ? <CircleCheck size={16} /> : <CircleX size={16} />}
      </Box>
    </Tooltip>
  )
}

export function NexusGoneBadge({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const nexusId = useProfiles((s) => {
    const profile = openProfileOf(s)
    return profile ? nexusIdOf(profile, mod) : 0
  })
  const page = useNexusPage(nexusId)
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
  const hit = useLastRun((s) => s.byId[mod.id])
  if (!hit || (hit.errors === 0 && hit.warnings === 0)) {
    return null
  }
  const errors = hit.errors ? errorsLabel(hit.errors) : ''
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
    <Box
      sx={{ display: 'flex', flexShrink: 0, gap: 0.4 }}
      aria-label={t`Last run: ${{ summary: text }}`}
    >
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
  // The reason, or null while unpinned: a primitive, so saving the profile re-renders only this mod's badge.
  const reason = useProfiles((s) => {
    const entry = entryOf(openProfileOf(s), mod.key)
    return entry?.pinned ? (entry.pinReason ?? '') : null
  })
  if (reason === null) {
    return null
  }
  const text = reason ? t`Pinned: ${{ pinned: reason }}` : t`Pinned at this version`
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
  const updates = useUpdates((s) => s.updates)
  // updateFor returns the stored update itself, so a profile save that leaves this mod's offer alone changes nothing.
  const update = useProfiles((s) => updateFor(updates, mod, openProfileOf(s)))
  const nexusId = useProfiles((s) => {
    const profile = openProfileOf(s)
    return profile ? nexusIdOf(profile, mod) : 0
  })
  const page = useNexusPage(nexusId)
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
          sx={{ p: 0, flexShrink: 0, color: 'var(--mortar-accent-ink)', borderRadius: '4px' }}
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
        <MenuAction
          icon={<Ban size={16} />}
          label={t`Skip this update`}
          onClick={() => {
            setAnchor(null)
            setSkipVersion(mod, update.version).catch(reportUnexpected)
          }}
        />
      </Menu>
    </>
  )
}

export function ModSwitch({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const setEnabled = useMods((s) => s.setEnabled)
  const locked = useLocked()
  // The switch moves at once; the list and the profile change together when the save returns, in one render.
  const [moving, setMoving] = useState<boolean | null>(null)
  const on = moving ?? mod.enabled
  return (
    <LockedReason locked={locked}>
      <Switch
        size="small"
        checked={on}
        disabled={locked}
        onChange={(e) => {
          setMoving(e.target.checked)
          setEnabled(mod, e.target.checked)
            .catch(reportUnexpected)
            .finally(() => setMoving(null))
        }}
        onClick={(e) => e.stopPropagation()}
        slotProps={{
          input: {
            'aria-label': on ? t`Disable ${mod.name}` : t`Enable ${mod.name}`,
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
      fullWidth={true}
      sx={footerButtonSx}
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
      <Button
        variant="outlined"
        fullWidth={true}
        disabled={locked}
        onClick={() => askRemove(mod)}
        sx={{ ...footerButtonSx, ...destructiveSx, borderColor: 'error.light' }}
      >
        {t`Remove`}
      </Button>
    </LockedReason>
  )
}

export function RemoveDialog() {
  const removing = useMods((s) => s.removing)
  const askRemove = useMods((s) => s.askRemove)
  // The body subscribes to the mod list and profile; a closed dialog renders none of it.
  return (
    <Dialog open={removing.length > 0} onClose={() => askRemove(null)}>
      <RemoveDialogBody removing={removing} />
    </Dialog>
  )
}
