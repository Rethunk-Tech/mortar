import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  alpha,
  Box,
  Button,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
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
  X,
} from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import type { Update } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import { download, type Want } from '../queue/actions.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { changelogsBetween, mergeCachedDetails } from './changelogRange.ts'
import { modId, sameId, siblingsOf, updateCount } from './lookup.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { accent, paper } from './paper.ts'
import { LetterTile } from './parts.tsx'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

const MONO = '"IBM Plex Mono", monospace'
const ROW_TILE = 48
const MEDIUM = 500
const DIALOG_WIDTH = 760
const ICON_TILE = 44
const ICON_FILL = 0.18

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

const downloadable = (u: Update) => u.githubRepo !== '' || u.nexusId > 0

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
          <Typography
            sx={{ fontSize: 13, pl: 2, whiteSpace: 'pre-line', overflowWrap: 'anywhere' }}
          >
            {(c.notes ?? []).map((n) => `• ${n}`).join('\n')}
          </Typography>
        </Box>
      ))}
    </Fold>
  )
}

function Row({ update, profileId }: { update: Update; profileId: string }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const mod = mods.find((m) => m.key === update.key && sameId(m.uniqueId, update.uniqueId))
  const queued = useQueue((s) => pendingUpdate(s.state.items, profileId, update))
  const notes = [
    ...(mod ? siblingsOf(mods, mod).map((o) => t`Also updates ${o.name} (same download)`) : []),
    ...(mod && !mod.enabled ? [t`Switched off in this profile`] : []),
  ]
  return (
    <Box
      role="listitem"
      sx={{
        display: 'grid',
        gridTemplateColumns: `${ROW_TILE}px minmax(0, 1fr) auto auto`,
        gap: '14px',
        alignItems: 'center',
        px: 3,
        py: 1.75,
        borderBottom: '1px solid rgba(255,255,255,0.08)',
      }}
    >
      <LetterTile mod={{ uniqueId: update.uniqueId, name: update.name }} size={ROW_TILE} />
      <Box sx={{ minWidth: 0, display: 'flex', flexDirection: 'column', gap: '3px' }}>
        <Typography sx={{ fontSize: 16, fontWeight: 600, overflowWrap: 'anywhere' }}>
          {update.name}
        </Typography>
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
          <Button
            variant="contained"
            disabled={queued}
            onClick={() => download([updateWant(update)]).catch(reportUnexpected)}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {queued ? t`Queued` : t`Update`}
          </Button>
        ) : null}
      </Box>
    </Box>
  )
}

export function UpdateBar() {
  const { t } = useLingui()
  const updates = useUpdates((s) => s.updates)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const count = updateCount(updates)
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
  const setReviewing = useUpdates((s) => s.setReviewing)
  const close = () => setReviewing(false)
  const list = updates?.updates ?? []
  const items = useQueue((s) => s.state.items)
  useEffect(() => {
    if (!open) {
      return
    }
    mergeCachedDetails(
      (updates?.updates ?? []).filter((u) => u.nexusId > 0).map((u) => u.nexusId),
    ).catch(reportUnexpected)
  }, [open, updates])
  const wanted = list
    .filter((u) => downloadable(u) && !pendingUpdate(items, profile.id, u))
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
            {updates?.unknown
              ? t`Some mods could not be checked, so more updates may show up later.`
              : t`Checked with SMAPI's update service`}
          </Typography>
        </Box>
        <IconButton aria-label={t`Close`} onClick={close}>
          <X size={16} />
        </IconButton>
      </DialogTitle>
      <DialogContent sx={{ p: 0, borderTop: '1px solid rgba(255,255,255,0.08)' }}>
        <Box role="list">
          {list.map((u) => (
            <Row key={modId(u)} update={u} profileId={profile.id} />
          ))}
        </Box>
      </DialogContent>
      <DialogActions sx={{ px: 3, py: 2, gap: 1.5, bgcolor: 'rgba(0,0,0,0.2)' }}>
        <Box sx={{ color: '#a3d3f7', display: 'flex' }}>
          <ShieldCheck size={18} aria-hidden={true} />
        </Box>
        <Typography sx={{ flex: 1, fontSize: 13, lineHeight: 1.45 }}>
          {t`Update downloads a mod's new file from Nexus or GitHub. For other pages, download the archive and drop it on this window: Mortar updates the mod in place, keeps its settings, and backs up your saves first. Roll back any mod later from its details.`}
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
    </Dialog>
  )
}
