import { useLingui } from '@lingui/react/macro'
import { Box, Link, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { modsLabel, problemsLabel, updatesLabel } from '../i18n/counts.ts'
import { useBadges } from '../mods/badges.ts'
import { unlinkCollection } from '../profiles/collectionUnlink.ts'
import { userModCount } from '../profiles/count.ts'
import { ProfileMark } from '../profiles/ProfileMark.tsx'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { openImport } from '../share/store.ts'
import { collectionHeader } from './collectionHeader.ts'
import { compact, compactMeta } from './compact.ts'
import { HeroCover } from './HeroCover.tsx'
import { NameField } from './NameField.tsx'
import { useRenameRequest } from './renameRequest.ts'
import { useCollectionStatus } from './useCollectionStatus.ts'

const MARK_SIZE = 44
const MARK_SIZE_COMPACT = 18
const NAME_FONT_PX = 44
const NAME_WEIGHT = 700
const NAME_LINE_HEIGHT = 1.1
const NAME_FONT_COMPACT_PX = 18
const NAME_LINE_COMPACT = 1.3
// The full hero always sits on game art or the dark band behind it, so its text is light in both themes; the
// compact strip is a theme surface and takes the theme's ink.
const ON_ART = '#ffffff'
const ON_ART_85 = 'rgba(255, 255, 255, 0.85)'
const NAME_GLOW = '0 1px 16px rgba(0, 0, 0, 0.6)'
const DESC_FONT_PX = 13
const DESC_SHADOW = '0 1px 8px rgba(0, 0, 0, 0.55)'
const DOT = '·'
const META_FONT_PX = 12
const HERO_HEIGHT_PX = 150
const HERO_COMPACT_HEIGHT_PX = 52
const HERO_COMPACT_BG = 'var(--mortar-hero)'
const HERO_COMPACT_BORDER = '1px solid var(--mortar-hairline)'
const COVER_TINT = 'var(--mortar-overlay-20)'
const HERO_INSET_PX = 24
const HERO_BOTTOM_PX = 16
const HERO_INSET_COMPACT_PX = 12

function CollectionLine({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const pending = useCollectionStatus(game, profile)
  const { line, review } = collectionHeader(pending)
  if (!line) {
    return null
  }
  return (
    <Typography
      noWrap={true}
      sx={{
        mt: 0.5,
        fontSize: META_FONT_PX + 1,
        color: ON_ART_85,
        textShadow: DESC_SHADOW,
        display: 'flex',
        alignItems: 'center',
        gap: 0.75,
      }}
    >
      <span>{t`From the Nexus collection “${line.name}”`}</span>
      <span>
        {review === null ? t`· revision ${line.revision}` : t`· revision ${review} is out`}
      </span>
      {review === null ? null : (
        <>
          <span>{DOT}</span>
          <Link
            component="button"
            underline="hover"
            onClick={() => openImport({ profileId: profile.id, collectionUpdate: true })}
          >
            {t`Review`}
          </Link>
        </>
      )}
      <span>{DOT}</span>
      <Link component="button" underline="hover" onClick={() => unlinkCollection(game, profile)}>
        {t`Unlink`}
      </Link>
    </Typography>
  )
}

function HeroName({
  profile,
  meta,
  game,
  compactAt,
}: {
  profile: Profile
  meta: string[]
  game: string
  compactAt: string
}) {
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
                <Box sx={{ flexShrink: 0, [compactAt]: { display: 'none' } }}>
                  <ProfileMark profile={profile} size={MARK_SIZE} />
                </Box>
                <Box sx={{ display: 'none', flexShrink: 0, [compactAt]: { display: 'block' } }}>
                  <ProfileMark profile={profile} size={MARK_SIZE_COMPACT} />
                </Box>
              </>
            ) : null}
            <Typography
              noWrap={true}
              title={profile.name}
              sx={{
                fontSize: NAME_FONT_PX,
                fontWeight: NAME_WEIGHT,
                lineHeight: NAME_LINE_HEIGHT,
                color: ON_ART,
                textShadow: NAME_GLOW,
                [compactAt]: {
                  fontSize: NAME_FONT_COMPACT_PX,
                  lineHeight: NAME_LINE_COMPACT,
                  color: 'var(--mortar-ink)',
                  textShadow: 'none',
                },
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
            fontSize: DESC_FONT_PX,
            color: ON_ART_85,
            textShadow: DESC_SHADOW,
            [compactAt]: { display: 'none' },
          }}
        >
          {profile.description}
        </Typography>
      ) : null}
      <Typography
        title={meta.join(' · ')}
        noWrap={true}
        sx={{ display: 'none', fontSize: META_FONT_PX, [compactAt]: { display: 'block' } }}
      >
        {meta.join(' · ')}
      </Typography>
      <CollectionLine profile={profile} game={game} />
    </Box>
  )
}

export function Hero({ profile, game }: { profile: Profile; game: string }) {
  const mods = userModCount(profile)
  const updates = useBadges((s) => s.byProfile[profile.id]?.updates ?? 0)
  const problems = useBadges((s) => s.byProfile[profile.id]?.problems ?? 0)
  const hero = useSettings((s) => s.profileHero) || 'full'
  const meta = compactMeta(mods, updates, problems).map((part) => {
    if (part.kind === 'mods') {
      return modsLabel(part.n)
    }
    if (part.kind === 'updates') {
      return updatesLabel(part.n)
    }
    return problemsLabel(part.n)
  })
  if (hero === 'hidden') {
    return null
  }
  const forceCompact = hero === 'compact'
  // The Compact setting applies the narrow-window layout at every width; '&' targets the element itself.
  const compactAt = forceCompact ? '&' : compact
  return (
    <Box
      sx={{
        position: 'relative',
        height: HERO_HEIGHT_PX,
        flexShrink: 0,
        overflow: 'hidden',
        borderBottom: HERO_COMPACT_BORDER,
        [compactAt]: {
          height: HERO_COMPACT_HEIGHT_PX,
          bgcolor: HERO_COMPACT_BG,
        },
      }}
    >
      <Box
        sx={{
          position: 'absolute',
          inset: 0,
          [compactAt]: { display: 'none' },
        }}
      >
        <HeroCover game={game} profile={profile} />
        <Box sx={{ position: 'absolute', inset: 0, bgcolor: COVER_TINT }} />
      </Box>
      <Box
        sx={{
          position: 'absolute',
          left: HERO_INSET_PX,
          right: HERO_INSET_PX,
          bottom: HERO_BOTTOM_PX,
          display: 'flex',
          alignItems: 'flex-end',
          gap: 2,
          [compactAt]: {
            top: 0,
            bottom: 0,
            left: HERO_INSET_COMPACT_PX,
            right: HERO_INSET_COMPACT_PX,
            alignItems: 'center',
          },
        }}
      >
        <HeroName profile={profile} meta={meta} game={game} compactAt={compactAt} />
      </Box>
    </Box>
  )
}
