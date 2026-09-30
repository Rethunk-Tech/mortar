import { plural } from '@lingui/core/macro'
import { Trans, useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Tab, Tabs, Typography } from '@mui/material'
import { Pencil, Plus } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { ModsTab } from '../mods/ModsTab.tsx'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { compact } from './compact.ts'
import { NameField } from './NameField.tsx'
import { NewProfileDialog } from './NewProfileDialog.tsx'

const fmt = (iso: unknown) => new Date(String(iso)).toLocaleDateString()

function Card({ label, value }: { label: string; value: string }) {
  return (
    <Box sx={{ px: 1.5, py: 0.75, bgcolor: 'rgba(28,28,32,0.6)', borderRadius: '6px' }}>
      <Typography sx={{ fontSize: 12, color: 'text.secondary', whiteSpace: 'nowrap' }}>
        {label}
      </Typography>
      <Typography sx={{ fontSize: 15, fontWeight: 600, whiteSpace: 'nowrap' }}>{value}</Typography>
    </Box>
  )
}

function Hero({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const art = useProfiles((s) => s.game?.artUrl)
  const rename = useProfiles((s) => s.rename)
  const band = useMods((s) => s.view) === 'list'
  const [editing, setEditing] = useState(false)
  const mods = (profile.entries ?? []).reduce((n, e) => n + (e.mods ?? []).length, 0)
  const created = fmt(profile.created)
  const updated = fmt(profile.updated)
  return (
    <Box
      sx={{
        position: 'relative',
        height: band ? 96 : 190,
        flexShrink: 0,
        overflow: 'hidden',
        bgcolor: 'rgba(15,15,18,0.5)',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'flex-end',
        gap: 1.5,
        px: 3,
        py: 2,
        [compact]: { height: 56, flexDirection: 'row', alignItems: 'center', gap: 2, py: 0 },
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
            opacity: 0.5,
            [compact]: { display: 'none' },
          }}
        />
      ) : null}
      <Box
        sx={{ position: 'relative', display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}
      >
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
                fontSize: band ? 28 : 40,
                fontWeight: 700,
                lineHeight: 1.2,
                textShadow: '0 0 18px rgba(255,255,255,0.45)',
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
      <Box
        sx={{
          position: 'relative',
          display: band ? 'none' : 'flex',
          gap: 1,
          [compact]: { display: 'none' },
        }}
      >
        <Card label={t`Created`} value={created} />
        <Card label={t`Updated`} value={updated} />
        <Card label={t`Mods`} value={String(mods)} />
      </Box>
      <Typography
        noWrap={true}
        sx={{
          position: 'relative',
          display: band ? 'block' : 'none',
          fontSize: 14,
          color: 'text.secondary',
          [compact]: { display: 'block' },
        }}
      >
        {t`Created ${created} · Updated ${updated}`} ·{' '}
        {plural(mods, { one: '# mod', other: '# mods' })}
      </Typography>
    </Box>
  )
}

export function Detail() {
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
        <Typography sx={{ fontSize: 24, fontWeight: 600 }}>
          <Trans>No profiles yet</Trans>
        </Typography>
        <Typography sx={{ color: 'text.secondary' }}>
          <Trans>A profile holds one set of mods for this game.</Trans>
        </Typography>
        <Button
          variant="contained"
          startIcon={<Plus size={16} />}
          onClick={() => setCreating(true)}
        >
          <Trans>Create your first profile</Trans>
        </Button>
        <NewProfileDialog open={creating} onClose={() => setCreating(false)} />
      </Box>
    )
  }
  return (
    <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
      <Hero key={profile.id} profile={profile} />
      <Tabs value="mods" sx={{ px: 2, flexShrink: 0 }}>
        <Tab value="mods" label={<Trans>Mods</Trans>} />
      </Tabs>
      <ModsTab key={profile.id} profile={profile} />
    </Box>
  )
}
