import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Tab, Tabs, Typography } from '@mui/material'
import { Pencil, Plus } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { ModsTab } from '../mods/ModsTab.tsx'
import { useProfiles } from '../profiles/store.ts'
import { compact } from './compact.ts'
import { NameField } from './NameField.tsx'
import { NewProfileDialog } from './NewProfileDialog.tsx'

const fmt = (iso: unknown) => new Date(String(iso)).toLocaleDateString()

function Card({ label, value }: { label: string; value: string }) {
  return (
    <Box
      sx={{
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
  const mods = (profile.entries ?? []).reduce((n, e) => n + (e.mods ?? []).length, 0)
  const created = fmt(profile.created)
  const updated = fmt(profile.updated)
  return (
    <Box
      sx={{
        position: 'relative',
        height: 190,
        flexShrink: 0,
        overflow: 'hidden',
        bgcolor: 'rgba(15,15,18,0.5)',
        [compact]: { height: 56 },
      }}
    >
      {art ? (
        <Box
          component="img"
          src={art}
          alt=""
          sx={{
            position: 'absolute',
            inset: 0,
            width: '100%',
            height: '100%',
            objectFit: 'cover',
            filter: 'blur(1px) brightness(0.75)',
            opacity: 0.6,
            [compact]: { display: 'none' },
          }}
        />
      ) : null}
      <Box
        sx={{
          position: 'absolute',
          inset: 0,
          bgcolor: 'rgba(20,20,24,0.50)',
          [compact]: { display: 'none' },
        }}
      />
      <Box
        sx={{
          position: 'absolute',
          left: 24,
          right: 24,
          bottom: 16,
          display: 'flex',
          alignItems: 'flex-end',
          gap: 2,
          [compact]: { top: 0, bottom: 0, left: 16, right: 16, alignItems: 'center' },
        }}
      >
        <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', alignItems: 'center', gap: 1 }}>
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
                  [compact]: { fontSize: 22 },
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
        <Box sx={{ display: 'flex', gap: 1, [compact]: { display: 'none' } }}>
          <Card label={t`Mods`} value={String(mods)} />
          <Card label={t`Updated`} value={updated} />
          <Card label={t`Created`} value={created} />
        </Box>
        <Typography
          noWrap={true}
          sx={{
            display: 'none',
            fontSize: 14,
            color: 'text.secondary',
            [compact]: { display: 'block' },
          }}
        >
          {t`Created ${created} · Updated ${updated} · ${plural(mods, { one: '# mod', other: '# mods' })}`}
        </Typography>
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
  const profile = profiles.find((p) => p.id === openId)
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
          value="mods"
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
        </Tabs>
      </Box>
      <ModsTab key={`mods-${profile.id}`} profile={profile} />
    </Box>
  )
}
