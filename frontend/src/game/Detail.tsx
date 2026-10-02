import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Chip,
  Divider,
  IconButton,
  Menu,
  Tab,
  Tabs,
  Tooltip,
  Typography,
} from '@mui/material'
import { MoreHorizontal, Palette, Pencil, Plus, RotateCcw, Settings2, Share2 } from 'lucide-react'
import { type ReactNode, useEffect, useRef, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { ConsoleTab } from '../console/ConsoleTab.tsx'
import { LogActions } from '../console/LogActions.tsx'
import { PerformancePanel } from '../console/PerformancePanel.tsx'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useBadges } from '../mods/badges.ts'
import { ModsTab } from '../mods/ModsTab.tsx'
import { ProblemActions, ProblemsTab } from '../mods/ProblemsTab.tsx'
import { problemCount } from '../mods/problemGroups.ts'
import { useMods } from '../mods/store.ts'
import { useNav } from '../nav/store.ts'
import { NotesTab } from '../notes/NotesTab.tsx'
import { userModCount } from '../profiles/count.ts'
import { EditProfileDialog } from '../profiles/EditProfileDialog.tsx'
import { HistoryDialog } from '../profiles/HistoryDialog.tsx'
import { ProfileMark } from '../profiles/ProfileMark.tsx'
import { useProfiles } from '../profiles/store.ts'
import { SavesTab } from '../saves/SavesTab.tsx'
import { useSaves } from '../saves/store.ts'
import { openShare } from '../share/store.ts'
import { IconAction } from '../shell/IconAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { ToolsMenu } from '../tools/ToolsMenu.tsx'
import { compact, compactMeta, saveFits } from './compact.ts'
import { CoverButton, HeroCover } from './HeroCover.tsx'
import { NameField } from './NameField.tsx'
import { NewProfileDialog } from './NewProfileDialog.tsx'
import { MoreMenuItems } from './ProfileMenuItems.tsx'
import { useRenameRequest } from './renameRequest.ts'
import { type TabId, useTab } from './tab.ts'
import { useRestoreFocus } from './useRestoreFocus.ts'

const fmt = (iso: unknown) => formatWhen(String(iso))

function HeroMenu({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [historyOpen, setHistoryOpen] = useState(false)
  const close = () => setAnchor(null)
  return (
    <>
      <Tooltip title={t`Profile menu`}>
        <IconButton
          aria-label={t`Profile menu`}
          aria-haspopup="menu"
          onClick={(e) => setAnchor(e.currentTarget)}
          size="small"
        >
          <MoreHorizontal size={16} />
        </IconButton>
      </Tooltip>
      {/* Kept mounted: the items own dialogs (Send, Compare, Delete) that must outlive the closed menu. */}
      <Menu
        open={anchor !== null}
        anchorEl={anchor}
        onClose={close}
        transitionDuration={0}
        keepMounted={true}
      >
        <MoreMenuItems profile={profile} close={close} onHistory={() => setHistoryOpen(true)} />
      </Menu>
      <HistoryDialog
        profileId={profile.id}
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
      />
    </>
  )
}

function Card({ label, value, onClick }: { label: string; value: string; onClick?: () => void }) {
  return (
    <Box
      component={onClick ? 'button' : 'div'}
      onClick={onClick}
      sx={{
        border: 0,
        color: 'inherit',
        font: 'inherit',
        textAlign: 'left',
        cursor: onClick ? 'pointer' : 'default',
        '&:hover': onClick ? { bgcolor: 'rgba(60,60,70,0.9)' } : undefined,
        display: 'flex',
        flexDirection: 'column',
        px: 1.5,
        py: 1,
        bgcolor: 'rgba(40,40,48,0.85)',
        borderRadius: '6px',
      }}
    >
      <Typography sx={{ fontSize: 12, color: 'text.secondary', whiteSpace: 'nowrap' }}>
        {label}
      </Typography>
      <Typography sx={{ fontSize: 18, fontWeight: 700, lineHeight: 1.4, whiteSpace: 'nowrap' }}>
        {value}
      </Typography>
    </Box>
  )
}

function HeroName({ profile, game, meta }: { profile: Profile; game: string; meta: string[] }) {
  const { t } = useLingui()
  const rename = useProfiles((s) => s.rename)
  const [editing, setEditing] = useState(false)
  const [editingProfile, setEditingProfile] = useState(false)
  const renameId = useRenameRequest((s) => s.id)
  useEffect(() => {
    if (renameId === profile.id) {
      useRenameRequest.getState().clear()
      setEditing(true)
    }
  }, [renameId, profile.id])
  const pencil = useRef<HTMLButtonElement>(null)
  useRestoreFocus(editing, pencil)
  return (
    <Box sx={{ flexGrow: 1, minWidth: 0 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        {editing ? (
          <NameField
            initial={profile.name}
            label={t`Profile name`}
            onSubmit={(name) =>
              name.trim() === profile.name ? Promise.resolve(true) : rename(profile.id, name)
            }
            onCancel={() => setEditing(false)}
          />
        ) : (
          <>
            {profile.color || profile.icon ? (
              <>
                <Box sx={{ flexShrink: 0, [compact]: { display: 'none' } }}>
                  <ProfileMark profile={profile} size={44} />
                </Box>
                <Box sx={{ display: 'none', flexShrink: 0, [compact]: { display: 'block' } }}>
                  <ProfileMark profile={profile} size={18} />
                </Box>
              </>
            ) : null}
            <Typography
              noWrap={true}
              sx={{
                fontSize: 44,
                fontWeight: 700,
                lineHeight: 1.1,
                color: '#ffffff',
                textShadow: '0 0 32px rgba(255,255,255,0.45)',
                [compact]: { fontSize: 18, lineHeight: 1.3 },
              }}
            >
              {profile.name}
            </Typography>
            <Tooltip title={t`Rename profile`}>
              <IconButton
                ref={pencil}
                aria-label={t`Rename profile`}
                onClick={() => setEditing(true)}
                size="small"
              >
                <Pencil size={16} />
              </IconButton>
            </Tooltip>
            <Tooltip title={t`Edit profile`}>
              <IconButton
                aria-label={t`Edit profile`}
                onClick={() => setEditingProfile(true)}
                size="small"
              >
                <Palette size={16} />
              </IconButton>
            </Tooltip>
            <CoverButton game={game} profile={profile} />
            <HeroMenu profile={profile} />
          </>
        )}
      </Box>
      {profile.description ? (
        <Typography
          noWrap={true}
          title={profile.description}
          sx={{
            mt: 0.5,
            fontSize: 13,
            color: 'rgba(255,255,255,0.72)',
            textShadow: '0 1px 8px rgba(0,0,0,0.55)',
            [compact]: { display: 'none' },
          }}
        >
          {profile.description}
        </Typography>
      ) : null}
      <Typography
        noWrap={true}
        sx={{ display: 'none', fontSize: 12, [compact]: { display: 'block' } }}
      >
        {meta.join(' · ')}
      </Typography>
      <EditProfileDialog
        profile={profile}
        open={editingProfile}
        onClose={() => setEditingProfile(false)}
      />
    </Box>
  )
}

function Hero({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const mods = userModCount(profile)
  const setTab = useTab((s) => s.setTab)
  const fits = useSaves((s) => s.fits)
  const { fitting, total } = saveFits(fits)
  const created = fmt(profile.created)
  const updated = fmt(profile.updated)
  const updates = useBadges((s) => s.byProfile[profile.id]?.updates ?? 0)
  const problems = useBadges((s) => s.byProfile[profile.id]?.problems ?? 0)
  const meta = compactMeta(mods, updates, problems).map((part) => {
    if (part.kind === 'mods') {
      return t`${plural(part.n, { one: '# mod', other: '# mods' })}`
    }
    if (part.kind === 'updates') {
      return t`${plural(part.n, { one: '# update', other: '# updates' })}`
    }
    return t`${plural(part.n, { one: '# problem', other: '# problems' })}`
  })
  return (
    <Box
      sx={{
        position: 'relative',
        height: 190,
        flexShrink: 0,
        overflow: 'hidden',
        [compact]: {
          height: 52,
          bgcolor: 'rgba(15,15,18,0.5)',
          borderBottom: '1px solid rgba(255,255,255,0.1)',
        },
      }}
    >
      <Box
        sx={{
          position: 'absolute',
          inset: 0,
          maskImage: HERO_FADE,
          WebkitMaskImage: HERO_FADE,
          [compact]: { display: 'none' },
        }}
      >
        <HeroCover game={game} profile={profile} />
        <Box sx={{ position: 'absolute', inset: 0, bgcolor: 'rgba(20,20,24,0.18)' }} />
      </Box>
      <Box
        sx={{
          position: 'absolute',
          left: 24,
          right: 24,
          bottom: 16,
          display: 'flex',
          alignItems: 'flex-end',
          gap: 2,
          [compact]: { top: 0, bottom: 0, left: 12, right: 12, alignItems: 'center' },
        }}
      >
        <HeroName profile={profile} game={game} meta={meta} />
        <Box sx={{ display: 'flex', gap: 1, [compact]: { display: 'none' } }}>
          <Card label={t`Mods`} value={String(mods)} />
          {total === 0 ? (
            <Tooltip title={t`No saves were found`}>
              <Card
                label={t`Saves`}
                value={t`${fitting} of ${total}`}
                onClick={() => setTab('saves')}
              />
            </Tooltip>
          ) : (
            <Card
              label={t`Saves`}
              value={t`${fitting} of ${total}`}
              onClick={() => setTab('saves')}
            />
          )}
          <Card label={t`Updated`} value={updated} />
          <Card label={t`Created`} value={created} />
        </Box>
      </Box>
    </Box>
  )
}

function Centered({ children }: { children: ReactNode }) {
  return (
    <Box
      sx={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 1.5,
        textAlign: 'center',
        px: 3,
      }}
    >
      {children}
    </Box>
  )
}

// The cover stays opaque until its bottom fifth, then fades to transparent into the backdrop above the tab row.
const HERO_FADE = 'linear-gradient(to bottom, #000 80%, transparent 100%)'

export function Detail() {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const openId = useProfiles((s) => s.openId)
  const loaded = useProfiles((s) => s.loaded)
  const failed = useProfiles((s) => s.failed)
  const load = useProfiles((s) => s.load)
  const openProfiles = useNav((s) => s.openProfiles)
  const [creating, setCreating] = useState(false)
  const tab = useTab((s) => s.tab)
  const setTab = useTab((s) => s.setTab)
  const problemsResult = useMods((s) => s.problems)
  const problemsTabCount = problemsResult === null ? null : problemCount(problemsResult)
  const game = useProfiles((s) => s.game?.id ?? '')
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const openGameSettings = useNav((s) => s.openGameSettings)
  const profile = profiles.find((p) => p.id === openId)
  const loadSaves = useSaves((s) => s.load)
  const profileId = profile?.id
  const updated = String(profile?.updated)
  // The profile's mods change with `updated`, and the save comparison follows them.
  useEffect(() => {
    if (game && profileId) {
      loadSaves(game, profileId, updated).catch(reportUnexpected)
    }
  }, [game, profileId, updated, loadSaves])
  if (!loaded) {
    return failed ? (
      <Centered>
        <Typography
          sx={{ fontSize: 24, fontWeight: 600 }}
        >{t`Could not read your profiles`}</Typography>
        <Button
          variant="contained"
          startIcon={<RotateCcw size={16} />}
          onClick={() => {
            load(failed).catch(reportUnexpected)
          }}
        >
          {t`Retry`}
        </Button>
      </Centered>
    ) : null
  }
  if (!profile && profiles.length > 0) {
    return (
      <Centered>
        <Typography sx={{ fontSize: 24, fontWeight: 600 }}>{t`All profiles are hidden`}</Typography>
        <Typography sx={{ color: 'text.secondary' }}>
          {t`Show one in the sidebar from Manage profiles.`}
        </Typography>
        <Button variant="contained" startIcon={<Settings2 size={16} />} onClick={openProfiles}>
          {t`Manage profiles`}
        </Button>
      </Centered>
    )
  }
  if (!profile) {
    return (
      <Centered>
        <Typography sx={{ fontSize: 24, fontWeight: 600 }}>{t`No profiles yet`}</Typography>
        <Typography sx={{ color: 'text.secondary' }}>
          {t`A profile holds one set of mods for this game.`}
        </Typography>
        <Button
          variant="contained"
          startIcon={<Plus size={16} />}
          onClick={() => setCreating(true)}
        >
          {t`Create your first profile`}
        </Button>
        <NewProfileDialog open={creating} onClose={() => setCreating(false)} />
      </Centered>
    )
  }
  return (
    <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
      <Hero key={`hero-${profile.id}`} profile={profile} game={game} />
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          px: 2,
          borderBottom: '1px solid rgba(255,255,255,0.1)',
          flexShrink: 0,
        }}
      >
        <Tabs
          value={tab}
          onChange={(_, value: TabId) => setTab(value)}
          sx={{
            minHeight: 44,
            '& .MuiTabs-indicator': { height: 2 },
            '& .MuiTab-root': {
              minHeight: 44,
              minWidth: 0,
              px: '14px',
              fontSize: 14,
              fontWeight: 400,
              color: 'text.secondary',
              '&.Mui-selected': { color: '#ffffff', fontWeight: 600 },
            },
          }}
        >
          <Tab value="mods" label={t`Mods`} />
          <Tab
            value="problems"
            label={
              <Box
                component="span"
                sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.75 }}
              >
                {t`Problems`}
                {problemsTabCount !== null && problemsTabCount > 0 ? (
                  <Chip
                    size="small"
                    label={problemsTabCount}
                    sx={{ height: 20, fontSize: 12, fontWeight: 600 }}
                  />
                ) : null}
              </Box>
            }
          />
          <Tab value="saves" label={t`Saves`} />
          <Tab value="notes" label={t`Notes`} />
          <Tab value="console" label={t`Console`} />
          <Tab value="performance" label={t`Performance`} />
        </Tabs>
        <Box sx={{ flexGrow: 1 }} />
        <Box sx={{ display: 'flex', gap: 0.75 }}>
          {tab === 'problems' ? (
            <>
              <ProblemActions />
              <Divider orientation="vertical" flexItem={true} sx={{ mx: 0.5 }} />
            </>
          ) : null}
          {tab === 'console' ? (
            <>
              <LogActions />
              <Divider orientation="vertical" flexItem={true} sx={{ mx: 0.5 }} />
            </>
          ) : null}
          <ToolsMenu game={game} profileID={profile.id} />
          <IconAction
            label={t`Share`}
            icon={<Share2 size={16} />}
            onClick={() => openShare(profile.id)}
          />
          <IconAction
            label={t`${gameName} settings`}
            icon={<Settings2 size={16} />}
            onClick={openGameSettings}
          />
        </Box>
      </Box>
      {tab === 'console' ? <ConsoleTab game={game} /> : null}
      {tab === 'performance' ? <PerformancePanel game={game} /> : null}
      {tab === 'notes' ? <NotesTab key={`notes-${profile.id}`} profile={profile} /> : null}
      {tab === 'saves' ? <SavesTab profile={profile} game={game} /> : null}
      {tab === 'mods' ? <ModsTab key={`mods-${profile.id}`} profile={profile} /> : null}
      {tab === 'problems' ? <ProblemsTab key={`problems-${profile.id}`} /> : null}
    </Box>
  )
}
