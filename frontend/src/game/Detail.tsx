import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Tab, Tabs, Typography } from '@mui/material'
import { Pencil, Plus, Settings2, Share2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { ConsoleTab, LogActions } from '../console/ConsoleTab.tsx'
import { ModsTab } from '../mods/ModsTab.tsx'
import { useNav } from '../nav/store.ts'
import { NotesTab } from '../notes/NotesTab.tsx'
import { userModCount } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { SavesTab } from '../saves/SavesTab.tsx'
import { useSaves } from '../saves/store.ts'
import { openShare } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { compact } from './compact.ts'
import { NameField } from './NameField.tsx'
import { NewProfileDialog } from './NewProfileDialog.tsx'
import { type TabId, useTab } from './tab.ts'

const fmt = (iso: unknown) => new Date(String(iso)).toLocaleDateString()

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

function Hero({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const art = useProfiles((s) => s.game?.artUrl)
  const rename = useProfiles((s) => s.rename)
  const [editing, setEditing] = useState(false)
  const mods = userModCount(profile)
  const setTab = useTab((s) => s.setTab)
  const fits = useSaves((s) => s.fits)
  const savesReady = useSaves((s) => s.status === 'ready')
  const fitting = fits.filter((f) => (f.missing ?? []).length === 0).length
  const total = fits.length
  const created = fmt(profile.created)
  const updated = fmt(profile.updated)
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
          maskImage: 'linear-gradient(to bottom, #000 60%, transparent 100%)',
          WebkitMaskImage: 'linear-gradient(to bottom, #000 60%, transparent 100%)',
          [compact]: { display: 'none' },
        }}
      >
        {art ? (
          <Box
            component="img"
            src={art}
            alt=""
            sx={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
          />
        ) : null}
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
                <IconButton
                  aria-label={t`Rename profile`}
                  onClick={() => setEditing(true)}
                  size="small"
                >
                  <Pencil size={16} />
                </IconButton>
              </>
            )}
          </Box>
          <Typography
            noWrap={true}
            sx={{ display: 'none', fontSize: 12, [compact]: { display: 'block' } }}
          >
            {t`${plural(mods, { one: '# mod', other: '# mods' })} · Updated ${updated}`}
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 1, [compact]: { display: 'none' } }}>
          <Card label={t`Mods`} value={String(mods)} />
          {savesReady && fits.length > 0 ? (
            <Card
              label={t`Saves`}
              value={t`${fitting} of ${total}`}
              onClick={() => setTab('saves')}
            />
          ) : null}
          <Card label={t`Updated`} value={updated} />
          <Card label={t`Created`} value={created} />
        </Box>
      </Box>
    </Box>
  )
}

export function Detail() {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const openId = useProfiles((s) => s.openId)
  const loaded = useProfiles((s) => s.loaded)
  const [creating, setCreating] = useState(false)
  const tab = useTab((s) => s.tab)
  const setTab = useTab((s) => s.setTab)
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
    return null
  }
  if (!profile) {
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
      </Box>
    )
  }
  return (
    <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
      <Hero key={`hero-${profile.id}`} profile={profile} />
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
              textTransform: 'none',
              color: 'text.secondary',
              '&.Mui-selected': { color: '#ffffff', fontWeight: 600 },
            },
          }}
        >
          <Tab value="mods" label={t`Mods`} />
          <Tab value="saves" label={t`Saves`} />
          <Tab value="notes" label={t`Notes`} />
          <Tab value="console" label={t`Console`} />
        </Tabs>
        <Box sx={{ flexGrow: 1 }} />
        {tab === 'console' ? <LogActions /> : null}
        <Button
          variant="outlined"
          color="inherit"
          startIcon={<Share2 size={16} />}
          onClick={() => openShare(profile.id)}
          sx={{ ml: 1, whiteSpace: 'nowrap' }}
        >
          {t`Share`}
        </Button>
        <IconButton aria-label={t`${gameName} settings`} onClick={openGameSettings} sx={{ ml: 1 }}>
          <Settings2 size={18} />
        </IconButton>
      </Box>
      {tab === 'console' ? <ConsoleTab game={game} /> : null}
      {tab === 'notes' ? <NotesTab key={`notes-${profile.id}`} profile={profile} /> : null}
      {tab === 'saves' ? <SavesTab profile={profile} game={game} /> : null}
      {tab === 'mods' ? <ModsTab key={`mods-${profile.id}`} profile={profile} /> : null}
    </Box>
  )
}
