import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  alpha,
  Box,
  Button,
  Checkbox,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Tooltip,
  Typography,
} from '@mui/material'
import { Browser } from '@wailsio/runtime'
import {
  ArrowRight,
  ArrowUp,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  ShieldCheck,
  TriangleAlert,
  X,
} from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import type { Update } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { UpdateEntry } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { download, type Want } from '../queue/actions.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import {
  changelogNoteIsRisky,
  changelogsBetween,
  changelogsHaveRiskyNotes,
  mergeCachedDetails,
} from './changelogRange.ts'
import {
  installableUpdate,
  listedAgainstNexus,
  modId,
  sameId,
  siblingsOf,
  updateCount,
  visibleUpdates,
} from './lookup.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { OtherProfilesDialog } from './OtherProfilesDialog.tsx'
import { accent, paper } from './paper.ts'
import { LetterTile } from './parts.tsx'
import { useMods } from './store.ts'
import { checkedWithSmapi, useUpdates } from './updates.ts'

const MONO = '"IBM Plex Mono", monospace'
const ROW_TILE = 48
const MEDIUM = 500
const DIALOG_WIDTH = 760
const ICON_TILE = 44
const ICON_FILL = 0.18
const TICK_MS = 60_000

function Version({ children, isNew }: { children: string; isNew?: boolean }) {
  return (
    <Box
      component="span"
      sx={{
        px: 1,
        py: '3px',
        borderRadius: '4px',
        fontFamily: MONO,
        fontSize: 13,
        fontWeight: isNew ? MEDIUM : 'normal',
        color: isNew ? 'primary.main' : 'text.primary',
        bgcolor: isNew ? accent.chip : 'rgba(255,255,255,0.08)',
      }}
    >
      {children}
    </Box>
  )
}

// A mod whose update lives on GitHub comes from there; otherwise the update is a Nexus file.
const updateWant = (u: Update): Want => ({
  kind: 'update',
  ...(u.githubRepo ? { repo: u.githubRepo } : { modId: u.nexusId }),
  name: u.name,
  version: u.version,
  currentKey: u.key,
})

const downloadable = (u: Update) => installableUpdate(u)

const installedCaution = (mods: Mod[], u: Update): string => {
  const mod = mods.find((m) => m.key === u.key && sameId(m.uniqueId, u.uniqueId))
  return mod?.updateCautionMessage?.trim() ?? ''
}

const pendingUpdate = (items: Item[], profileId: string, u: Update) =>
  u.githubRepo
    ? pendingFor(items, profileId, 0, u.githubRepo)
    : pendingFor(items, profileId, u.nexusId)

function Fold({ title, children }: { title: string; children: ReactNode }) {
  const [shown, setShown] = useState(false)
  const Icon = shown ? ChevronDown : ChevronRight
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Button
        size="small"
        onClick={() => setShown(!shown)}
        startIcon={<Icon size={14} aria-hidden={true} />}
        aria-expanded={shown}
        sx={{
          whiteSpace: 'nowrap',
          fontSize: 12,
          color: 'text.secondary',
          fontWeight: 700,
          textTransform: 'none',
          alignSelf: 'flex-start',
          px: 0.5,
        }}
      >
        {title}
      </Button>
      <Collapse in={shown} unmountOnExit={true}>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, pl: 1 }}>{children}</Box>
      </Collapse>
    </Box>
  )
}

function Changes({ update }: { update: Update }) {
  const { t } = useLingui()
  const details = useNexusDetails((s) => s.byId[update.nexusId]?.details)
  if (!update.nexusId) {
    return null
  }
  if (!details) {
    return (
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
        {t`Changelog is not cached yet.`}
      </Typography>
    )
  }
  const all = details.changelogs ?? []
  const logs = changelogsBetween(all, update.installed, update.version)
  if (all.length === 0) {
    return (
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
        {t`This mod has no changelog.`}
      </Typography>
    )
  }
  if (logs.length === 0) {
    return (
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
        {t`No changelog entries between these versions.`}
      </Typography>
    )
  }
  return (
    <Fold
      title={t`${plural(logs.length, { one: '# version of changes', other: '# versions of changes' })}`}
    >
      {logs.map((c) => (
        <Box key={c.version}>
          <Typography sx={{ fontSize: 13, fontWeight: 700 }}>{c.version}</Typography>
          {(c.notes ?? []).map((n) => (
            <Typography
              key={n}
              sx={{
                fontSize: 13,
                pl: 2,
                whiteSpace: 'pre-line',
                overflowWrap: 'anywhere',
                color: changelogNoteIsRisky(n) ? 'warning.main' : undefined,
              }}
            >
              {`• ${n}`}
            </Typography>
          ))}
        </Box>
      ))}
    </Fold>
  )
}

function Row({
  update,
  profileId,
  caution,
  acked,
  onAck,
  onUpdateAll,
}: {
  update: Update
  profileId: string
  caution: string
  acked: boolean
  onAck: (on: boolean) => void
  onUpdateAll: () => void
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const mod = mods.find((m) => m.key === update.key && sameId(m.uniqueId, update.uniqueId))
  const details = useNexusDetails((s) => s.byId[update.nexusId]?.details)
  const riskyChangelog =
    update.nexusId > 0 && details
      ? changelogsHaveRiskyNotes(
          changelogsBetween(details.changelogs ?? [], update.installed, update.version),
        )
      : false
  const queued = useQueue((s) => pendingUpdate(s.state.items, profileId, update))
  const notes = [
    ...(mod ? siblingsOf(mods, mod).map((o) => t`Also updates ${o.name} (same download)`) : []),
    ...(mod && !mod.enabled ? [t`Switched off in this profile`] : []),
    ...(update.unofficial ? [t`Unofficial`] : []),
  ]
  return (
    <Box
      role="listitem"
      sx={{
        display: 'grid',
        gridTemplateColumns: caution
          ? `${ROW_TILE}px minmax(0, 1fr) auto auto auto`
          : `${ROW_TILE}px minmax(0, 1fr) auto auto`,
        gap: '14px',
        alignItems: 'center',
        px: 3,
        py: 1.75,
        borderBottom: '1px solid rgba(255,255,255,0.08)',
      }}
    >
      <LetterTile mod={{ uniqueId: update.uniqueId, name: update.name }} size={ROW_TILE} />
      <Box sx={{ minWidth: 0, display: 'flex', flexDirection: 'column', gap: '3px' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, minWidth: 0 }}>
          <Typography sx={{ fontSize: 16, fontWeight: 600, overflowWrap: 'anywhere', minWidth: 0 }}>
            {update.unofficial ? t`Unofficial update available: ${update.version}` : update.name}
          </Typography>
          {riskyChangelog ? (
            <Tooltip title={t`This update's notes mention breaking changes or new requirements`}>
              <Box
                component="span"
                sx={{ display: 'inline-flex', flexShrink: 0, color: 'warning.main' }}
              >
                <TriangleAlert size={16} aria-hidden={true} />
              </Box>
            </Tooltip>
          ) : null}
        </Box>
        {notes.length > 0 ? (
          <Typography
            sx={{
              alignSelf: 'flex-start',
              px: 1,
              borderRadius: '10px',
              bgcolor: 'rgba(255,255,255,0.08)',
              fontSize: 12,
            }}
          >
            {notes.join(' · ')}
          </Typography>
        ) : null}
        {caution ? (
          <Typography sx={{ fontSize: 12, color: 'warning.main', overflowWrap: 'anywhere' }}>
            {caution}
          </Typography>
        ) : null}
        <Changes update={update} />
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, whiteSpace: 'nowrap' }}>
        <Version>{update.installed}</Version>
        <ArrowRight size={14} aria-hidden={true} />
        <Version isNew={true}>{update.version}</Version>
      </Box>
      <Box sx={{ display: 'flex', gap: 1 }}>
        {update.url ? (
          <Button
            variant="outlined"
            endIcon={<ExternalLink size={12} />}
            onClick={() => Browser.OpenURL(update.url).catch(reportUnexpected)}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Open page`}
          </Button>
        ) : null}
        {downloadable(update) ? (
          <>
            <Button
              variant="contained"
              disabled={queued || (caution !== '' && !acked)}
              onClick={() => download([updateWant(update)]).catch(reportUnexpected)}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {queued ? t`Queued` : t`Update`}
            </Button>
            <Button
              variant="outlined"
              disabled={queued || (caution !== '' && !acked)}
              onClick={onUpdateAll}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {t`Update in all profiles that have it`}
            </Button>
          </>
        ) : null}
      </Box>
      {caution ? (
        <Checkbox
          checked={acked}
          onChange={(_, on) => onAck(on)}
          slotProps={{ input: { 'aria-label': t`Confirm update for ${update.name}` } }}
          sx={{ justifySelf: 'center' }}
        />
      ) : null}
    </Box>
  )
}

// PropagateUpdate offers the update just applied to this profile to the game's other profiles that hold the old
// file, once the queue has installed it here and the new entry key is known.
function PropagateUpdate({
  profile,
  update,
  onDone,
}: {
  profile: Profile
  update: Update
  onDone: () => void
}) {
  const { t } = useLingui()
  const items = useQueue((s) => s.state.items)
  const [newKey, setNewKey] = useState('')
  useEffect(() => {
    const done = items.find(
      (item) =>
        item.state === 'done' &&
        item.kind === 'update' &&
        item.profileId === profile.id &&
        item.name === update.name &&
        item.version === update.version,
    )
    if (!done) {
      return
    }
    const next = useProfiles
      .getState()
      .profiles.find((candidate) => candidate.id === profile.id)
      ?.entries?.find((entry) =>
        entry.mods?.some((mod) => sameId(mod.uniqueId, update.uniqueId)),
      )?.key
    if (next) {
      setNewKey(next)
    }
  }, [items, profile.id, update])
  return (
    <OtherProfilesDialog
      open={newKey !== ''}
      onClose={onDone}
      game={useProfiles.getState().game?.id ?? ''}
      currentProfileId={profile.id}
      uniqueId={update.uniqueId}
      title={t`Update ${update.name} in other profiles`}
      confirmLabel={t`Update profiles`}
      update={{ oldKey: update.key }}
      onConfirm={async (profiles, pinned) => {
        const game = useProfiles.getState().game?.id ?? ''
        await Promise.all(profiles.map((other) => UpdateEntry(game, other.id, update.key, newKey)))
        useToasts.getState().push({
          kind: 'success',
          title: t`Updated in ${profiles.length} profiles`,
          ...(pinned.length > 0
            ? { body: pinned.map((other) => t`pinned in ${other.name}`).join(', ') }
            : {}),
        })
      }}
    />
  )
}

export function UpdateBar() {
  const { t } = useLingui()
  const updates = useUpdates((s) => s.updates)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const count = updateCount(updates, profile)
  if (count === 0) {
    return updates?.unknown ? (
      <Typography sx={{ mx: 2, mt: 1, fontSize: 12, color: 'text.secondary' }}>
        {t`Updates are unknown: SMAPI's update service could not be reached.`}
      </Typography>
    ) : null
  }
  return (
    <Box
      sx={{
        mx: 2,
        mt: 1.25,
        minHeight: 38,
        flexShrink: 0,
        display: 'flex',
        alignItems: 'center',
        gap: 1.25,
        pl: 1.5,
        pr: 0.75,
        fontSize: 14,
        bgcolor: accent.fill,
        border: '1px solid',
        borderColor: accent.line,
        borderRadius: '6px',
      }}
    >
      <Box component="span" sx={{ display: 'flex', flexShrink: 0, color: 'primary.main' }}>
        <ArrowUp size={16} aria-hidden={true} />
      </Box>
      <Typography noWrap={true} sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>
        {t`${plural(count, { one: '# update is available', other: '# updates are available' })}`}
      </Typography>
      <Button
        size="small"
        variant="contained"
        onClick={() => setReviewing(true)}
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Review`}
      </Button>
    </Box>
  )
}

export function UpdateReview({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const open = useUpdates((s) => s.reviewing)
  const updates = useUpdates((s) => s.updates)
  const checkedAt = useUpdates((s) => s.checkedAt)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const close = () => setReviewing(false)
  const byId = useNexusDetails((s) => s.byId)
  const list = visibleUpdates(updates, profile).filter((u) =>
    listedAgainstNexus(u, byId[u.nexusId]?.details?.page),
  )
  const items = useQueue((s) => s.state.items)
  const mods = useMods((s) => s.mods)
  const [acked, setAcked] = useState<Record<string, boolean>>({})
  const [now, setNow] = useState(() => Date.now())
  const [propagating, setPropagating] = useState<Update | null>(null)
  useEffect(() => {
    const id = globalThis.setInterval(() => setNow(Date.now()), TICK_MS)
    return () => globalThis.clearInterval(id)
  }, [])
  useEffect(() => {
    if (!open) {
      return
    }
    mergeCachedDetails(
      (updates?.updates ?? []).filter((u) => u.nexusId > 0).map((u) => u.nexusId),
    ).catch(reportUnexpected)
  }, [open, updates])
  const cautionOk = (u: Update) => {
    const caution = installedCaution(mods, u)
    return caution === '' || acked[modId(u)] === true
  }
  const wanted = list
    .filter((u) => downloadable(u) && !pendingUpdate(items, profile.id, u) && cautionOk(u))
    .map(updateWant)
  return (
    <Dialog
      open={open && list.length > 0}
      onClose={close}
      transitionDuration={0}
      maxWidth={false}
      slotProps={{
        paper: { sx: { ...paper.sx, width: DIALOG_WIDTH, maxWidth: 'calc(100% - 32px)' } },
      }}
    >
      <DialogTitle component="div" sx={{ display: 'flex', alignItems: 'center', gap: 1.75, p: 3 }}>
        <Box
          sx={{
            width: ICON_TILE,
            height: ICON_TILE,
            display: 'grid',
            placeItems: 'center',
            borderRadius: '10px',
            color: 'primary.main',
            bgcolor: (th) => alpha(th.palette.primary.main, ICON_FILL),
          }}
        >
          <ArrowUp size={22} aria-hidden={true} />
        </Box>
        <Box sx={{ flexGrow: 1 }}>
          <Typography component="h2" sx={{ fontSize: 22, fontWeight: 700 }}>
            {t`${plural(list.length, { one: '# update', other: '# updates' })} for ${profile.name}`}
          </Typography>
          <Typography sx={{ fontSize: 13 }}>
            {checkedWithSmapi(checkedAt, now, updates?.unknown === true)}
          </Typography>
        </Box>
        <IconButton aria-label={t`Close`} onClick={close}>
          <X size={16} />
        </IconButton>
      </DialogTitle>
      <DialogContent sx={{ p: 0, borderTop: '1px solid rgba(255,255,255,0.08)' }}>
        <Box role="list">
          {list.map((u) => {
            const caution = installedCaution(mods, u)
            const id = modId(u)
            return (
              <Row
                key={id}
                update={u}
                profileId={profile.id}
                caution={caution}
                acked={acked[id] === true}
                onAck={(on) => setAcked((prev) => ({ ...prev, [id]: on }))}
                onUpdateAll={() => {
                  download([updateWant(u)])
                    .then((added) => {
                      if (added) {
                        setPropagating(u)
                      }
                    })
                    .catch(reportUnexpected)
                }}
              />
            )
          })}
        </Box>
      </DialogContent>
      <DialogActions sx={{ px: 3, py: 2, gap: 1.5, bgcolor: 'rgba(0,0,0,0.2)' }}>
        <Box sx={{ color: '#a3d3f7', display: 'flex' }}>
          <ShieldCheck size={18} aria-hidden={true} />
        </Box>
        <Typography sx={{ flex: 1, fontSize: 13, lineHeight: 1.45 }}>
          {t`Update downloads a mod's new file from Nexus or GitHub. For other pages, download the archive and drop it on this window: Mortar updates the mod in place, keeps its settings, and backs up your saves first. Roll back any mod later from its details.`}
          <br />
          {t`Mortar checks for updates at startup, when you press F5, and at most once an hour while it is running.`}
        </Typography>
        <Button variant="outlined" onClick={close} sx={{ whiteSpace: 'nowrap' }}>
          {t`Close`}
        </Button>
        {wanted.length > 0 ? (
          <Button
            variant="contained"
            onClick={() => {
              close()
              download(wanted, true).catch(reportUnexpected)
            }}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Update all`}
          </Button>
        ) : null}
      </DialogActions>
      {propagating ? (
        <PropagateUpdate
          profile={profile}
          update={propagating}
          onDone={() => setPropagating(null)}
        />
      ) : null}
    </Dialog>
  )
}
