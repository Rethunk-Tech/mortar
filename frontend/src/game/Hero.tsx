import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Typography } from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { CollectionStatus } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import { CollectionStatus as loadCollectionStatus } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/service.ts'
import { When } from '../i18n/When.tsx'
import { useBadges } from '../mods/badges.ts'
import { openPage } from '../mods/menu.ts'
import { userModCount } from '../profiles/count.ts'
import { ProfileMark } from '../profiles/ProfileMark.tsx'
import { useProfiles } from '../profiles/store.ts'
import { useSaves } from '../saves/store.ts'
import { useSettings } from '../settings/store.ts'
import { openImport } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { collectionHeader } from './collectionHeader.ts'
import { compact, compactMeta, saveFits } from './compact.ts'
import { HeroCover } from './HeroCover.tsx'
import { NameField } from './NameField.tsx'
import { useRenameRequest } from './renameRequest.ts'
import { useTab } from './tab.ts'

const CARD_HOVER = 'rgba(60,60,70,0.9)'
const CARD_BG = 'rgba(40,40,48,0.85)'
const CARD_RADIUS = '6px'
const LABEL_FONT_PX = 12
const VALUE_FONT_PX = 18
const VALUE_LINE_HEIGHT = 1.4
const MARK_SIZE = 44
const MARK_SIZE_COMPACT = 18
const NAME_FONT_PX = 44
const NAME_WEIGHT = 700
const NAME_LINE_HEIGHT = 1.1
const NAME_FONT_COMPACT_PX = 18
const NAME_LINE_COMPACT = 1.3
const NAME_GLOW = '0 0 32px rgba(255,255,255,0.45)'
const DESC_FONT_PX = 13
const DESC_COLOR = 'rgba(255,255,255,0.72)'
const DESC_SHADOW = '0 1px 8px rgba(0,0,0,0.55)'
const META_FONT_PX = 12
const HERO_HEIGHT_PX = 190
const HERO_COMPACT_HEIGHT_PX = 52
const HERO_COMPACT_BG = 'rgba(15,15,18,0.5)'
const HERO_COMPACT_BORDER = '1px solid rgba(255,255,255,0.1)'
const COVER_TINT = 'rgba(20,20,24,0.18)'
const HERO_INSET_PX = 24
const HERO_BOTTOM_PX = 16
const HERO_INSET_COMPACT_PX = 12

function Card({
  label,
  value,
  onClick,
  ariaLabel,
}: {
  label: string
  value: ReactNode
  onClick?: () => void
  ariaLabel?: string
}) {
  return (
    <Box
      component={onClick ? 'button' : 'div'}
      aria-label={onClick ? ariaLabel : undefined}
      onClick={onClick}
      sx={{
        border: 0,
        color: 'inherit',
        font: 'inherit',
        textAlign: 'left',
        cursor: onClick ? 'pointer' : 'default',
        '&:hover': onClick ? { bgcolor: CARD_HOVER } : undefined,
        display: 'flex',
        flexDirection: 'column',
        px: 1.5,
        py: 1,
        bgcolor: CARD_BG,
        borderRadius: CARD_RADIUS,
      }}
    >
      <Typography sx={{ fontSize: LABEL_FONT_PX, color: 'text.secondary', whiteSpace: 'nowrap' }}>
        {label}
      </Typography>
      <Typography
        sx={{
          fontSize: VALUE_FONT_PX,
          fontWeight: NAME_WEIGHT,
          lineHeight: VALUE_LINE_HEIGHT,
          whiteSpace: 'nowrap',
        }}
      >
        {value}
      </Typography>
    </Box>
  )
}

function CollectionLine({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const [status, setStatus] = useState<CollectionStatus | null>(null)
  useEffect(() => {
    if (!profile.collection) {
      setStatus(null)
      return
    }
    let cancelled = false
    loadCollectionStatus(game, profile.id)
      .then((next) => {
        if (!cancelled) {
          setStatus(next)
        }
      })
      .catch(() => {
        if (!cancelled && profile.collection) {
          setStatus({
            linked: true,
            name: profile.collection.name,
            revision: profile.collection.revision,
            latest: 0,
            url: `https://www.nexusmods.com/games/${profile.collection.domain}/collections/${profile.collection.slug}`,
          })
        }
      })
    return () => {
      cancelled = true
    }
  }, [game, profile.id, profile.collection])
  const pending =
    profile.collection && !status
      ? {
          linked: true,
          name: profile.collection.name,
          revision: profile.collection.revision,
          latest: 0,
          url: `https://www.nexusmods.com/games/${profile.collection.domain}/collections/${profile.collection.slug}`,
        }
      : status
  const { line, review } = collectionHeader(pending)
  if (!line) {
    return null
  }
  return (
    <Typography
      noWrap={true}
      sx={{ mt: 0.5, fontSize: META_FONT_PX, display: 'flex', alignItems: 'center', gap: 1 }}
    >
      <Link
        href={line.url}
        underline="hover"
        color="inherit"
        onClick={(e) => {
          e.preventDefault()
          openPage(line.url).catch(reportUnexpected)
        }}
      >
        {t`From collection ${line.name}, revision ${line.revision}`}
      </Link>
      {review === null ? null : (
        <Button
          size="small"
          color="warning"
          onClick={() => openImport({ profileId: profile.id, collectionUpdate: true })}
        >
          {t`Revision ${review} is out — Review`}
        </Button>
      )}
    </Typography>
  )
}

function HeroName({ profile, meta, game }: { profile: Profile; meta: string[]; game: string }) {
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
                  <ProfileMark profile={profile} size={MARK_SIZE} />
                </Box>
                <Box sx={{ display: 'none', flexShrink: 0, [compact]: { display: 'block' } }}>
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
                color: '#ffffff',
                textShadow: NAME_GLOW,
                [compact]: { fontSize: NAME_FONT_COMPACT_PX, lineHeight: NAME_LINE_COMPACT },
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
            color: DESC_COLOR,
            textShadow: DESC_SHADOW,
            [compact]: { display: 'none' },
          }}
        >
          {profile.description}
        </Typography>
      ) : null}
      <Typography
        noWrap={true}
        sx={{ display: 'none', fontSize: META_FONT_PX, [compact]: { display: 'block' } }}
      >
        {meta.join(' · ')}
      </Typography>
      <CollectionLine profile={profile} game={game} />
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
  const hero = useSettings((s) => s.profileHero) || 'full'
  const meta = compactMeta(mods, updates, problems).map((part) => {
    if (part.kind === 'mods') {
      return t`${plural(part.n, { one: '# mod', other: '# mods' })}`
    }
    if (part.kind === 'updates') {
      return t`${plural(part.n, { one: '# update', other: '# updates' })}`
    }
    return t`${plural(part.n, { one: '# problem', other: '# problems' })}`
  })
  if (hero === 'hidden') {
    return null
  }
  const forceCompact = hero === 'compact'
  return (
    <Box
      sx={{
        position: 'relative',
        height: HERO_HEIGHT_PX,
        flexShrink: 0,
        overflow: 'hidden',
        ...(forceCompact
          ? {
              height: HERO_COMPACT_HEIGHT_PX,
              bgcolor: HERO_COMPACT_BG,
              borderBottom: HERO_COMPACT_BORDER,
            }
          : {
              [compact]: {
                height: HERO_COMPACT_HEIGHT_PX,
                bgcolor: HERO_COMPACT_BG,
                borderBottom: HERO_COMPACT_BORDER,
              },
            }),
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
          [compact]: {
            top: 0,
            bottom: 0,
            left: HERO_INSET_COMPACT_PX,
            right: HERO_INSET_COMPACT_PX,
            alignItems: 'center',
          },
        }}
      >
        <HeroName profile={profile} meta={meta} game={game} />
        <Box sx={{ display: 'flex', gap: 1, [compact]: { display: 'none' } }}>
          <Card label={t`Mods`} value={String(mods)} />
          <Card
            label={t`Saves`}
            value={total === 0 ? t`None` : t`${fitting} of ${total}`}
            ariaLabel={t`Open saves (${fitting} of ${total})`}
            onClick={() => setTab('saves')}
          />
          <Card label={t`Updated`} value={<When value={profile.updated} />} />
          <Card label={t`Created`} value={<When value={profile.created} />} />
        </Box>
      </Box>
    </Box>
  )
}
