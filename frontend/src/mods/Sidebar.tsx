import { useLingui } from '@lingui/react/macro'
import { Box, Button, Drawer, Tooltip, Typography, useMediaQuery } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  Mod,
  ModInProfile,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { ProfilesWithMod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { compactQuery } from '../game/compact.ts'
import { openModInProfile } from '../profiles/findMod.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDescribe } from './describe.ts'
import { useDetail } from './detail.ts'
import {
  concerns,
  entryOf,
  kindLabel,
  modId,
  nexusIdOf,
  problemsOf,
  siblingsOf,
  sourceKind,
  updateFor,
} from './lookup.ts'
import { ModDependencyTree } from './ModDependencyTree.tsx'
import { ModNoteTags } from './ModNoteTags.tsx'
import { useLookedSnapshot, useNexusEntry, useNexusFresh } from './nexusDetails.ts'
import { formatCount, formatDate, isNewer } from './nexusFormat.ts'
import { accent, heading } from './paper.ts'
import { LetterTile, ModSwitch, RemoveButton, ShowFilesButton } from './parts.tsx'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

const noWrap = { whiteSpace: 'nowrap' } as const

function Field({ label, value }: { label: string; value: string }) {
  return (
    <Box>
      <Typography sx={heading}>{label}</Typography>
      <Typography sx={{ fontSize: 13, overflowWrap: 'anywhere' }}>{value}</Typography>
    </Box>
  )
}

// A value cut to its lines, the whole of it in a tooltip.
function Clipped({ label, value, lines = 1, accented = false }: ClippedProps) {
  return (
    <Box sx={{ minWidth: 0 }}>
      <Typography sx={heading}>{label}</Typography>
      <Tooltip title={value} placement="left">
        <Typography
          sx={{
            fontSize: 13,
            color: accented ? 'primary.main' : undefined,
            overflow: 'hidden',
            display: '-webkit-box',
            WebkitBoxOrient: 'vertical',
            WebkitLineClamp: lines,
            overflowWrap: 'anywhere',
          }}
        >
          {value}
        </Typography>
      </Tooltip>
    </Box>
  )
}

interface ClippedProps {
  label: string
  value: string
  lines?: number
  accented?: boolean
}

// What the cached Nexus page adds; nothing until the details arrive, so the rest of the panel never waits on them.
function NexusFields({ mod, nexusId }: { mod: Mod; nexusId: number }) {
  const { t, i18n } = useLingui()
  const details = useNexusEntry(nexusId)?.details
  useLookedSnapshot(nexusId, details)
  if (!details) {
    return null
  }
  const { page, category } = details
  return (
    <>
      {page.summary ? <Clipped label={t`Summary`} value={page.summary} lines={2} /> : null}
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 1.25 }}>
        <Clipped
          label={t`Latest on Nexus`}
          value={page.version || '—'}
          accented={isNewer(page.version, mod.version)}
        />
        <Clipped label={t`Category`} value={category || '—'} />
        <Clipped label={t`Downloads`} value={formatCount(page.downloads, i18n.locale)} />
        <Clipped label={t`Updated`} value={formatDate(page.updated, i18n.locale) || '—'} />
      </Box>
    </>
  )
}

function UpdateBanner({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const update = useUpdates((s) => updateFor(s.updates, mod, profile))
  const setReviewing = useUpdates((s) => s.setReviewing)
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  if (!update) {
    return null
  }
  return (
    <Box
      sx={{
        px: 1.5,
        py: 0.75,
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        borderRadius: '6px',
        bgcolor: accent.fill,
        border: '1px solid',
        borderColor: accent.line,
      }}
    >
      <Typography sx={{ flex: 1, fontSize: 13 }}>
        {t`Update available: ${update.installed} → ${update.version}`}
      </Typography>
      <Button
        size="small"
        variant="outlined"
        onClick={() => setSkipVersion(mod, update.version).catch(reportUnexpected)}
        sx={noWrap}
      >
        {t`Skip this update`}
      </Button>
      <Button size="small" variant="contained" onClick={() => setReviewing(true)} sx={noWrap}>
        {t`Update`}
      </Button>
    </Box>
  )
}

function ProblemLine({ mod }: { mod: Mod }) {
  const describe = useDescribe()
  const result = useMods((s) => s.problems)
  const mine = problemsOf(result).filter((p) => concerns(p, mod))
  if (mine.length === 0) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', gap: 1, color: 'warning.main' }}>
      <TriangleAlert size={16} style={{ flexShrink: 0, marginTop: 2 }} aria-hidden={true} />
      <Typography sx={{ fontSize: 13 }}>{mine.map(describe).join(' ')}</Typography>
    </Box>
  )
}

function AlsoInProfiles({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id)
  const [rows, setRows] = useState<ModInProfile[]>([])
  useEffect(() => {
    if (!game) {
      return
    }
    ProfilesWithMod(game, mod.uniqueId)
      .then((list) => setRows((list ?? []).filter((r) => r.profileId !== profile.id)))
      .catch(reportUnexpected)
  }, [game, mod.uniqueId, profile.id])
  if (rows.length === 0) {
    return null
  }
  return (
    <Box>
      <Typography sx={heading}>{t`Also in these profiles`}</Typography>
      {rows.map((r) => (
        <Button
          key={r.profileId}
          onClick={() =>
            openModInProfile({ profileId: r.profileId, key: r.key, uniqueId: r.uniqueId })
          }
          sx={{
            ...noWrap,
            display: 'block',
            width: '100%',
            justifyContent: 'flex-start',
            textAlign: 'left',
            textTransform: 'none',
            fontSize: 13,
            px: 0.5,
          }}
        >
          {`${r.profileName} · ${r.version} · ${r.enabled ? t`Enabled` : t`Switched off`}`}
        </Button>
      ))}
    </Box>
  )
}

function Inspector({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { t } = useLingui()
  const all = useMods((s) => s.mods)
  const setPinned = useMods((s) => s.setPinned)
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  const others = siblingsOf(all, mod)
  const setOpen = useDetail((s) => s.setOpen)
  const kind = sourceKind(profile, mod)
  const nexusId = nexusIdOf(profile, mod)
  const fresh = useNexusFresh(nexusId)
  const entry = entryOf(profile, mod.key)
  const offered = useUpdates((s) => updateFor(s.updates, mod, profile))
  const source = kindLabel(kind, {
    archive: t`Archive`,
    nexus: t`Nexus Mods`,
    github: t`GitHub`,
  })
  return (
    <Box sx={{ p: 1.75, display: 'flex', flexDirection: 'column', gap: 1.25, minHeight: '100%' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <LetterTile mod={mod} size={52} fresh={fresh} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography sx={{ fontSize: 16, fontWeight: 700, overflowWrap: 'anywhere' }}>
            {mod.name}
          </Typography>
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
            {`${mod.author} · ${source}`}
          </Typography>
        </Box>
        <ModSwitch mod={mod} />
      </Box>
      <Field label={t`Version`} value={mod.version} />
      <Button
        variant="outlined"
        onClick={() => setPinned(mod, !entry?.pinned).catch(reportUnexpected)}
        sx={noWrap}
      >
        {entry?.pinned ? t`Unpin` : t`Pin this version`}
      </Button>
      {entry?.skipVersion && !offered ? (
        <Button
          variant="outlined"
          onClick={() => setSkipVersion(mod, '').catch(reportUnexpected)}
          sx={noWrap}
        >
          {t`Show skipped update`}
        </Button>
      ) : null}
      <Field label={t`UniqueID`} value={mod.uniqueId} />
      {mod.endorsements > 0 ? (
        <Field label={t`Endorsements`} value={mod.endorsements.toLocaleString()} />
      ) : null}
      {nexusId ? <NexusFields key={nexusId} mod={mod} nexusId={nexusId} /> : null}
      <UpdateBanner mod={mod} />
      <ProblemLine mod={mod} />
      <ModDependencyTree mod={mod} />
      <AlsoInProfiles mod={mod} profile={profile} />
      <ModNoteTags profile={profile} mod={mod} />
      {others.length > 0 ? (
        <Box>
          <Typography sx={heading}>{t`In the same download`}</Typography>
          {others.map((o) => (
            <Typography key={o.uniqueId} sx={{ fontSize: 13 }}>
              {o.name}
            </Typography>
          ))}
        </Box>
      ) : null}
      <Box sx={{ flexGrow: 1 }} />
      <Button variant="contained" onClick={() => setOpen(true)} sx={noWrap}>
        {t`More details`}
      </Button>
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
          gap: 1,
        }}
      >
        <ShowFilesButton mod={mod} />
        <RemoveButton mod={mod} />
      </Box>
    </Box>
  )
}

// The one details sidebar of both views: an aside at the window's normal width, a right drawer below 960px.
export function ModSidebar({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const narrow = useMediaQuery(compactQuery)
  const mods = useMods((s) => s.mods)
  const detailId = useDetail((s) => s.detailId)
  const show = useDetail((s) => s.show)
  const selected = mods.find((m) => modId(m) === detailId)
  if (narrow) {
    return (
      <Drawer
        anchor="right"
        open={selected !== undefined}
        onClose={() => show(null)}
        sx={{ top: 'var(--title-bar)' }}
        slotProps={{
          paper: {
            sx: {
              width: 320,
              top: 'var(--title-bar)',
              height: 'calc(100% - var(--title-bar))',
              bgcolor: 'rgba(40,40,48,0.92)',
            },
          },
        }}
      >
        {selected ? <Inspector mod={selected} profile={profile} /> : null}
      </Drawer>
    )
  }
  return (
    <Box
      aria-label={t`Selected mod`}
      component="aside"
      sx={{
        width: 300,
        overflowY: 'auto',
        bgcolor: 'rgba(40,40,48,0.72)',
        borderLeft: '1px solid rgba(255,255,255,0.1)',
      }}
    >
      {selected ? (
        <Inspector mod={selected} profile={profile} />
      ) : (
        <Typography sx={{ p: 1.75, fontSize: 13, color: 'text.secondary' }}>
          {t`Select a mod to see its details.`}
        </Typography>
      )}
    </Box>
  )
}
