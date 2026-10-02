import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { When } from '../i18n/When.tsx'
import { useBadges } from '../mods/badges.ts'
import { userModCount } from '../profiles/count.ts'
import { ProfileMark } from '../profiles/ProfileMark.tsx'
import { useProfiles } from '../profiles/store.ts'
import { useSaves } from '../saves/store.ts'
import { compact, compactMeta, saveFits } from './compact.ts'
import { HeroCover } from './HeroCover.tsx'
import { NameField } from './NameField.tsx'
import { useRenameRequest } from './renameRequest.ts'
import { useTab } from './tab.ts'

function Card({
  label,
  value,
  onClick,
}: {
  label: string
  value: ReactNode
  onClick?: () => void
}) {
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

function HeroName({ profile, meta }: { profile: Profile; meta: string[] }) {
  const { t } = useLingui()
  const rename = useProfiles((s) => s.rename)
  const [editing, setEditing] = useState(false)
  const renameId = useRenameRequest((s) => s.id)
  useEffect(() => {
    if (renameId === profile.id) {
      useRenameRequest.getState().clear()
      setEditing(true)
    }
  }, [renameId, profile.id])
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
              title={profile.name}
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
    </Box>
  )
}

const HERO_FADE = 'linear-gradient(to bottom, #000 80%, transparent 100%)'

export function Hero({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const mods = userModCount(profile)
  const setTab = useTab((s) => s.setTab)
  const fits = useSaves((s) => s.fits)
  const { fitting, total } = saveFits(fits)
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
        <HeroName profile={profile} meta={meta} />
        <Box sx={{ display: 'flex', gap: 1, [compact]: { display: 'none' } }}>
          <Card label={t`Mods`} value={String(mods)} />
          <Card
            label={t`Saves`}
            value={total === 0 ? t`None` : t`${fitting} of ${total}`}
            onClick={() => setTab('saves')}
          />
          <Card label={t`Updated`} value={<When value={profile.updated} />} />
          <Card label={t`Created`} value={<When value={profile.created} />} />
        </Box>
      </Box>
    </Box>
  )
}
