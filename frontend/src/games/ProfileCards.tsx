import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import type { MouseEvent } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { Played } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  SetLastGame,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { orderProfiles } from '../game/profileOrder.ts'
import { When } from '../i18n/When.tsx'
import { useBadges } from '../mods/badges.ts'
import { ProfileHealth } from '../mods/ProfileHealth.tsx'
import { type GameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { errorDetails, errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { profileCardLastPlayedIso } from './profileCardLastPlayed.ts'
import { profileCardsSlice } from './profileCardsSlice.ts'
import { useProfileCardsMeta } from './useProfileCardsMeta.ts'

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

  const openProfile = (ev: MouseEvent, profileId: string) => {
    ev.stopPropagation()
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

  const openGame = (ev: MouseEvent) => {
    ev.stopPropagation()
    SetLastGame(gameId).catch((err: unknown) => fail(t`Could not save the last game`, err))
    useNav.getState().openGame(gameId)
  }

  if (visible.length === 0) {
    return null
  }

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'row',
        flexWrap: 'nowrap',
        alignItems: 'center',
        gap: '10px',
        mt: '6px',
        minHeight: 0,
        maxHeight: 28,
        overflow: 'hidden',
      }}
    >
      {visible.map((profile) => (
        <ProfileCard
          key={profile.id}
          gameId={gameId}
          profile={profile}
          lastPlayedAt={profileCardLastPlayedIso(profile.id, lastPlayed, runStarted)}
          onOpen={(ev) => openProfile(ev, profile.id)}
        />
      ))}
      {more > 0 ? (
        <ButtonBase
          onClick={openGame}
          sx={{
            flexShrink: 0,
            maxHeight: 28,
            px: 0.75,
            overflow: 'hidden',
            color: 'inherit',
            fontFamily: 'inherit',
            fontSize: 13,
            fontWeight: 600,
            whiteSpace: 'nowrap',
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
  onOpen,
}: {
  gameId: GameId
  profile: Profile
  lastPlayedAt: string
  onOpen: (ev: MouseEvent) => void
}) {
  const counts = useBadges((s) => s.byProfile[profile.id])
  return (
    <ButtonBase
      onClick={onOpen}
      sx={{
        display: 'flex',
        flexDirection: 'row',
        flexWrap: 'nowrap',
        alignItems: 'center',
        gap: 0.75,
        flexShrink: 0,
        maxHeight: 28,
        minWidth: 0,
        overflow: 'hidden',
        color: 'inherit',
        fontFamily: 'inherit',
        textAlign: 'left',
      }}
    >
      <Typography
        noWrap={true}
        title={profile.name}
        sx={{ fontSize: 13, fontWeight: 700, lineHeight: 1.2 }}
      >
        {profile.name}
      </Typography>
      <ProfileHealth counts={counts} game={gameId} profileId={profile.id} />
      {lastPlayedAt ? (
        <Box
          sx={{
            fontSize: 12,
            lineHeight: 1.2,
            overflow: 'hidden',
            whiteSpace: 'nowrap',
            opacity: 0.9,
          }}
        >
          <When value={lastPlayedAt} />
        </Box>
      ) : null}
    </ButtonBase>
  )
}

export { ProfileCards }
