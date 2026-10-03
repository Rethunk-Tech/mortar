import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Typography } from '@mui/material'
import { Play } from 'lucide-react'
import type { MouseEvent } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { Played } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  SetLastGame,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { orderProfiles } from '../game/profileOrder.ts'
import { When } from '../i18n/When.tsx'
import { playDirect } from '../launch/directPref.ts'
import { useLaunch } from '../launch/store.ts'
import { useBadges } from '../mods/badges.ts'
import { ProfileHealth } from '../mods/ProfileHealth.tsx'
import { type GameId, useNav } from '../nav/store.ts'
import { ProfileMark } from '../profiles/ProfileMark.tsx'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { errorDetails, errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { profileCardLastPlayedIso } from './profileCardLastPlayed.ts'
import { profileCardsSlice } from './profileCardsSlice.ts'
import { useProfileCardsMeta } from './useProfileCardsMeta.ts'

const CARD_MIN = 168
const GRID_GAP = 10
const PLAY_ICON = 18

function ProfileCards({
  gameId,
  profiles,
  lastPlayed,
}: {
  gameId: GameId
  profiles: Profile[]
  lastPlayed: Played | undefined
}) {
  const { t } = useLingui()
  const profileOrder = useSettings((s) => s.profileOrder)
  const start = useLaunch((s) => s.start)
  const starting = useLaunch((s) => s.starting)
  const ordered = orderProfiles(
    profiles.filter((p) => !p.hidden),
    profileOrder,
    lastPlayed?.profile ?? '',
  )
  const { visible, more } = profileCardsSlice(ordered)
  const runStarted = useProfileCardsMeta(gameId, profiles, visible, lastPlayed)

  const fail = (title: string, err: unknown) => {
    useToasts
      .getState()
      .push({ kind: 'error', title, body: errorMessage(err), detail: errorDetails(err) })
  }

  const openProfile = (profileId: string) => {
    SetLastGame(gameId).catch((err: unknown) => fail(t`Could not save the last game`, err))
    SetLastProfile(gameId, profileId).catch((err: unknown) =>
      fail(t`Could not save the open profile`, err),
    )
    useNav.getState().openGame(gameId)
    useProfiles
      .getState()
      .load(gameId)
      .then(() => useProfiles.getState().open(profileId))
      .catch((err: unknown) => fail(t`Could not read your profiles`, err))
  }

  const openProfilesPage = () => {
    useNav.setState({ route: { name: 'profiles', game: gameId } })
    useProfiles
      .getState()
      .load(gameId)
      .catch((err: unknown) => fail(t`Could not read your profiles`, err))
  }

  const playProfile = (ev: MouseEvent, profileId: string) => {
    ev.stopPropagation()
    SetLastGame(gameId).catch((err: unknown) => fail(t`Could not save the last game`, err))
    SetLastProfile(gameId, profileId).catch((err: unknown) =>
      fail(t`Could not save the open profile`, err),
    )
    useNav.getState().openGame(gameId)
    start(gameId, profileId, playDirect()).then(() => undefined)
  }

  if (visible.length === 0) {
    return null
  }

  return (
    <Box
      sx={{
        position: 'relative',
        flexShrink: 0,
        px: '96px',
        py: 1.25,
        display: 'grid',
        gridTemplateColumns: `repeat(auto-fill, minmax(${CARD_MIN}px, 1fr))`,
        gap: `${GRID_GAP}px`,
        borderTop: '1px solid rgba(0,0,0,0.45)',
        bgcolor: 'rgba(0,0,0,0.22)',
      }}
    >
      {visible.map((profile) => (
        <ProfileCard
          key={profile.id}
          gameId={gameId}
          profile={profile}
          lastPlayedAt={profileCardLastPlayedIso(profile.id, lastPlayed, runStarted)}
          playDisabled={starting}
          onOpen={() => openProfile(profile.id)}
          onPlay={(ev) => playProfile(ev, profile.id)}
        />
      ))}
      {more > 0 ? (
        <ButtonBase
          onClick={openProfilesPage}
          sx={{
            minHeight: 88,
            borderRadius: '8px',
            border: '1px dashed',
            borderColor: 'var(--mortar-hairline-24)',
            color: 'text.secondary',
            fontFamily: 'inherit',
            fontSize: 15,
            fontWeight: 600,
          }}
        >
          {t`+${more} more`}
        </ButtonBase>
      ) : null}
    </Box>
  )
}

function ProfileCard({
  gameId,
  profile,
  lastPlayedAt,
  playDisabled,
  onOpen,
  onPlay,
}: {
  gameId: GameId
  profile: Profile
  lastPlayedAt: string
  playDisabled: boolean
  onOpen: () => void
  onPlay: (ev: MouseEvent) => void
}) {
  const { t } = useLingui()
  const counts = useBadges((s) => s.byProfile[profile.id])
  const updates = counts?.updates ?? 0
  const updatesLine = updates > 0 ? plural(updates, { one: '# update', other: '# updates' }) : ''
  return (
    <ButtonBase
      onClick={onOpen}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'stretch',
        gap: 0.75,
        p: 1.25,
        minHeight: 88,
        borderRadius: '8px',
        textAlign: 'left',
        bgcolor: 'var(--mortar-paper-78)',
        border: '1px solid var(--mortar-hairline-12)',
        fontFamily: 'inherit',
        color: 'var(--mortar-ink)',
        '&:hover': { bgcolor: 'var(--mortar-raised)' },
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, minWidth: 0 }}>
        <ProfileMark profile={profile} size={24} />
        <Typography
          noWrap={true}
          title={profile.name}
          sx={{ flex: 1, fontSize: 14, fontWeight: 700 }}
        >
          {profile.name}
        </Typography>
        <ProfileHealth counts={counts} game={gameId} profileId={profile.id} />
      </Box>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 1,
          minHeight: 28,
        }}
      >
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 0.5,
            minWidth: 0,
            fontSize: 12,
            color: 'text.secondary',
          }}
        >
          {updatesLine ? (
            <Typography component="span" sx={{ fontSize: 12 }}>
              {updatesLine}
            </Typography>
          ) : null}
          {lastPlayedAt ? <When value={lastPlayedAt} /> : null}
        </Box>
        <Button
          type="button"
          variant="contained"
          size="small"
          disabled={playDisabled}
          aria-label={t`Play ${profile.name}`}
          startIcon={<Play size={PLAY_ICON} fill="currentColor" />}
          onClick={onPlay}
          sx={{
            flexShrink: 0,
            minWidth: 0,
            px: 1.25,
            py: 0.5,
            borderRadius: '6px',
            fontSize: 13,
            fontWeight: 700,
            textTransform: 'none',
            boxShadow: 'none',
          }}
        >
          {t`Play`}
        </Button>
      </Box>
    </ButtonBase>
  )
}

export { ProfileCards }
