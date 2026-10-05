import { useLingui } from '@lingui/react/macro'
import { Box, Button, Drawer, Link, Tooltip, Typography, useMediaQuery } from '@mui/material'
import { TriangleAlert, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  Mod,
  ModInProfile,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  CopyMods,
  ProfilesWithMod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { compactQuery } from '../game/compact.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { openModInProfile } from '../profiles/findMod.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { IconAction } from '../shell/IconAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { AuthorLink } from './AuthorLink.tsx'
import { useCustomCategories } from './customCategories.ts'
import { localId } from './dependents.ts'
import { useDescribe } from './describe.ts'
import { useDetail } from './detail.ts'
import { customCategoryById, resolvedCategoryLabel } from './group.ts'
import { HiddenInside } from './HiddenInside.tsx'
import { lastRunSummary, showLastRunInConsole, useLastRun } from './lastRun.ts'
import {
  concerns,
  entryOf,
  kindLabel,
  modId,
  nexusIdOf,
  problemsOf,
  sameId,
  siblingsOf,
  sourceKind,
  updateFor,
  visibleUpdates,
} from './lookup.ts'
import { ModDependencyTree } from './ModDependencyTree.tsx'
import { ModNoteTags } from './ModNoteTags.tsx'
import { ModUpdateControls } from './ModUpdateControls.tsx'
import { openPage } from './menu.ts'
import { useLookedSnapshot, useNexusEntry, useNexusFresh } from './nexusDetails.ts'
import { formatCount, isNewer } from './nexusFormat.ts'
import { goneCaption, nexusPageMark, offersNexusDownload } from './nexusMark.ts'
import { OptionalFiles } from './OptionalFiles.tsx'
import { OtherProfilesDialog } from './OtherProfilesDialog.tsx'
import { accent, heading } from './paper.ts'
import { LetterTile, ModSwitch, RemoveButton, ShowFilesButton } from './parts.tsx'
import { useMods } from './store.ts'
import { EverywhereDialog } from './updateReview/EverywhereDialog.tsx'
import { useUpdates } from './updates.ts'

const noWrap = { whiteSpace: 'nowrap' } as const

function Field({ label, value, hint }: { label: string; value: string; hint?: string }) {
  const caption = (
    <Typography component={hint ? 'span' : 'p'} sx={heading}>
      {label}
    </Typography>
  )
  return (
    <Box>
      <Tooltip title={hint ?? ''} disableHoverListener={!hint}>
        {caption}
      </Tooltip>
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
function NexusFields({
  mod,
  nexusId,
  categoryOverride,
}: {
  mod: Mod
  nexusId: number
  categoryOverride: string
}) {
  const { t, i18n } = useLingui()
  const customById = customCategoryById(useCustomCategories((s) => s.categories))
  const details = useNexusEntry(nexusId)?.details
  useLookedSnapshot(nexusId, details)
  if (!details) {
    return null
  }
  const { page, category } = details
  const gone = nexusPageMark(page.status, page.available, page.updated, page.created)
  const goneLabel = goneCaption(gone, {
    hidden: t`Hidden on Nexus`,
    hiddenDated: t`Hidden on Nexus · ${gone.date}`,
    removed: t`Removed from Nexus`,
    removedDated: t`Removed from Nexus · ${gone.date}`,
  })
  return (
    <>
      {goneLabel ? (
        <Typography sx={{ fontSize: 13, color: 'warning.main' }}>{goneLabel}</Typography>
      ) : null}
      {page.summary ? <Clipped label={t`Summary`} value={page.summary} lines={2} /> : null}
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 1.25 }}>
        <Clipped
          label={t`Latest on Nexus`}
          value={page.version || '—'}
          accented={isNewer(page.version, mod.version)}
        />
        <Clipped
          label={t`Category`}
          value={resolvedCategoryLabel(categoryOverride, category, customById) || '—'}
        />
        <Clipped label={t`Downloads`} value={formatCount(page.downloads, i18n.locale)} />
        <Clipped label={t`Updated`} value={formatWhen(page.updated) || '—'} />
      </Box>
    </>
  )
}

function UpdateBanner({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const profile = useProfiles(openProfileOf)
  const nexusId = profile ? nexusIdOf(profile, mod) : 0
  const details = useNexusEntry(nexusId)?.details
  const raw = useUpdates((s) => s.updates)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const game = useProfiles((s) => s.game?.id ?? '')
  const [everywhere, setEverywhere] = useState(false)
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  const mine = visibleUpdates(raw, profile).filter((u) => u.key === mod.key && sameId(u.id, mod.id))
  const unofficial = mine.find((u) => u.unofficial)
  const update = mine.find((u) => !u.unofficial)
  const blocked =
    Boolean(update) &&
    (update?.nexusId ?? 0) > 0 &&
    !offersNexusDownload(details?.page.status, details?.page.available)
  const official = blocked ? undefined : update
  if (!(official || unofficial)) {
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
      <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 0.25 }}>
        {official ? (
          <Typography sx={{ fontSize: 13 }}>
            {t`Update available: ${official.installed} → ${official.version}`}
          </Typography>
        ) : null}
        {unofficial ? (
          <Typography sx={{ fontSize: 13 }}>
            {t`Unofficial update available: ${unofficial.version}`}
            {unofficial.url ? (
              <>
                {' '}
                <Link
                  component="button"
                  onClick={() => openPage(unofficial.url)}
                  sx={{ fontSize: 13 }}
                >
                  {t`Open page`}
                </Link>
              </>
            ) : null}
          </Typography>
        ) : null}
      </Box>
      {official ? (
        <Button
          size="small"
          variant="outlined"
          onClick={() => setSkipVersion(mod, official.version).catch(reportUnexpected)}
        >
          {t`Skip this update`}
        </Button>
      ) : null}
      <Button size="small" variant="contained" onClick={() => setReviewing(true)}>
        {t`Update`}
      </Button>
      <Button size="small" variant="outlined" onClick={() => setEverywhere(true)}>
        {t`Update in all profiles that have it`}
      </Button>
      <EverywhereDialog
        open={everywhere}
        game={game}
        mods={[{ id: mod.id, newKey: 'latest' }]}
        onClose={() => setEverywhere(false)}
      />
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

function LastRunLine({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id)
  const hit = useLastRun((s) => s.byId[mod.id])
  if (!hit || (hit.errors === 0 && hit.warnings === 0) || !game) {
    return null
  }
  const summary = lastRunSummary(hit)
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75, alignItems: 'flex-start' }}>
      <Typography sx={{ fontSize: 13 }}>{t`Last run: ${summary}`}</Typography>
      <Button
        size="small"
        variant="outlined"
        onClick={() => showLastRunInConsole(game, profile.id, mod)}
      >
        {t`Show in Console`}
      </Button>
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
    let live = true
    ProfilesWithMod(game, mod.id)
      .then((list) => {
        if (live) {
          setRows((list ?? []).filter((r) => r.profileId !== profile.id))
        }
      })
      .catch(reportUnexpected)

    return () => {
      live = false
    }
  }, [game, mod.id, profile.id])
  if (rows.length === 0) {
    return null
  }
  return (
    <Box>
      <Typography sx={heading}>{t`Also in these profiles`}</Typography>
      {rows.map((r) => (
        <Button
          key={r.profileId}
          onClick={() => openModInProfile({ profileId: r.profileId, key: r.key, id: r.id })}
          title={`${r.profileName} · ${r.version} · ${r.enabled ? t`Enabled` : t`Off`}`}
          sx={{
            ...noWrap,
            display: 'block',
            width: '100%',
            justifyContent: 'flex-start',
            textAlign: 'left',
            fontSize: 13,
            px: 0.5,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {`${r.profileName} · ${r.version} · ${r.enabled ? t`Enabled` : t`Off`}`}
        </Button>
      ))}
    </Box>
  )
}

function Inspector({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { t, i18n } = useLingui()
  const all = useMods((s) => s.mods)
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  const setSkipSource = useMods((s) => s.setSkipSource)
  const others = siblingsOf(all, mod)
  const setOpen = useDetail((s) => s.setOpen)
  const game = useProfiles((s) => s.game?.id ?? '')
  const [alsoOpen, setAlsoOpen] = useState(false)
  const kind = sourceKind(profile, mod)
  const nexusId = nexusIdOf(profile, mod)
  const fresh = useNexusFresh(nexusId)
  const entry = entryOf(profile, mod.key)
  const offered = useUpdates((s) => updateFor(s.updates, mod, profile))
  const sourceName = kindLabel(kind, {
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
          <Typography
            component="div"
            sx={{ fontSize: 13, color: 'text.secondary', overflowWrap: 'anywhere' }}
          >
            <AuthorLink authorField={mod.author} mod={mod} profile={profile} />
            {` · ${sourceName}`}
          </Typography>
        </Box>
        <ModSwitch mod={mod} />
        <IconAction
          label={t`Close details`}
          icon={<X size={18} />}
          onClick={() => useDetail.getState().show(null)}
        />
      </Box>
      <Field label={t`Version`} value={mod.version} />
      <ModUpdateControls mod={mod} entry={entry} />
      {entry?.skipVersion && !offered ? (
        <Button variant="outlined" onClick={() => setSkipVersion(mod, '').catch(reportUnexpected)}>
          {t`Show skipped update`}
        </Button>
      ) : null}
      {(entry?.skipSources ?? []).map((ignoredSource) => (
        <Button
          key={ignoredSource}
          variant="outlined"
          onClick={() => setSkipSource(mod, ignoredSource, false).catch(reportUnexpected)}
        >
          {t`Stop ignoring ${ignoredSource} updates`}
        </Button>
      ))}
      <Field label={t`Mod id`} value={localId(mod.id)} hint={t`SMAPI UniqueID`} />
      {mod.endorsements > 0 ? (
        <Field label={t`Endorsements`} value={mod.endorsements.toLocaleString(i18n.locale)} />
      ) : null}
      {nexusId ? (
        <NexusFields
          key={nexusId}
          mod={mod}
          nexusId={nexusId}
          categoryOverride={entry?.categoryOverride ?? ''}
        />
      ) : null}
      <UpdateBanner mod={mod} />
      <OptionalFiles mod={mod} profile={profile} />
      <Button variant="outlined" onClick={() => setAlsoOpen(true)}>
        {t`Also add to…`}
      </Button>
      <OtherProfilesDialog
        open={alsoOpen}
        onClose={() => setAlsoOpen(false)}
        game={game}
        currentProfileId={profile.id}
        id={mod.id}
        title={t`Also add ${mod.name} to…`}
        confirmLabel={t`Add`}
        onConfirm={async (profiles) => {
          await Promise.all(profiles.map((other) => CopyMods(game, profile.id, other.id, [mod.id])))
        }}
      />
      <ProblemLine mod={mod} />
      <LastRunLine mod={mod} profile={profile} />
      <ModDependencyTree mod={mod} />
      <AlsoInProfiles mod={mod} profile={profile} />
      <HiddenInside mod={mod} profile={profile} />
      <ModNoteTags profile={profile} mod={mod} />
      {others.length > 0 ? (
        <Box>
          <Typography sx={heading}>{t`In the same download`}</Typography>
          {others.map((o) => (
            <Link
              key={o.id}
              component="button"
              onClick={() => useDetail.getState().show(o)}
              sx={{ display: 'block', fontSize: 13, textAlign: 'left', color: 'text.primary' }}
            >
              {o.name}
            </Link>
          ))}
        </Box>
      ) : null}
      <Box sx={{ flexGrow: 1 }} />
      <Button variant="contained" onClick={() => setOpen(true)}>
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
            role: 'dialog',
            'aria-label': t`Mod details`,
            sx: {
              width: 320,
              top: 'var(--title-bar)',
              height: 'calc(100% - var(--title-bar))',
              bgcolor: 'var(--mortar-panel-92)',
            },
          },
        }}
      >
        {selected ? <Inspector mod={selected} profile={profile} /> : null}
      </Drawer>
    )
  }
  // The details column only takes room while a mod is selected; the list gets the full width otherwise.
  if (!selected) {
    return null
  }
  return (
    <Box
      aria-label={t`Selected mod`}
      component="aside"
      sx={{
        width: 300,
        overflowY: 'auto',
        bgcolor: 'var(--mortar-panel)',
        borderLeft: '1px solid var(--mortar-hairline)',
      }}
    >
      <Inspector mod={selected} profile={profile} />
    </Box>
  )
}
