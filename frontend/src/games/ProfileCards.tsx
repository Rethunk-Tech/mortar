import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import type { MouseEvent } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Played } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { SetLastProfile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { orderProfiles } from '../game/profileOrder.ts'
import { When } from '../i18n/When.tsx'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { profileCardLastPlayedIso } from './profileCardLastPlayed.ts'
import { profileCardsSlice } from './profileCardsSlice.ts'
import { useOpenGame } from './useOpenGame.ts'
import { useProfileCardsMeta } from './useProfileCardsMeta.ts'

function ProfileCards({
  game,
  profiles,
  lastPlayed,
}: {
  game: GameInfo
  profiles: Profile[]
  lastPlayed: Played | undefined
}) {
  const { t } = useLingui()
  const profileOrder = useSettings((s) => s.profileOrder)
  const ordered = orderProfiles(profiles, profileOrder, lastPlayed?.profile ?? '')
  const { visible, more } = profileCardsSlice(ordered)
  const runStarted = useProfileCardsMeta(game.id, visible, lastPlayed)
  const open = useOpenGame()

  const openProfile = (ev: MouseEvent, profileId: string) => {
    ev.stopPropagation()
    SetLastProfile(game.id, profileId).catch(reportError(t`Could not save the open profile`))
    open(game)
    useProfiles.getState().load(game.id, profileId).catch(reportError(t`Could not read profiles`))
  }

  const onOpenGame = (ev: MouseEvent) => {
    ev.stopPropagation()
    open(game)
  }

  if (visible.length === 0) {
    return null
  }

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'row',
        // Chips wrap onto further lines rather than being cut off; the card limit keeps that to two or three.
        flexWrap: 'wrap',
        alignItems: 'center',
        gap: space.gap,
        mt: space.gap,
        maxWidth: '100%',
      }}
    >
      {visible.map((profile) => (
        <ProfileCard
          key={profile.id}
          profile={profile}
          lastPlayedAt={profileCardLastPlayedIso(profile.id, lastPlayed, runStarted)}
          onOpen={(ev) => openProfile(ev, profile.id)}
        />
      ))}
      {more > 0 ? (
        <ButtonBase
          aria-label={t`Open ${{ game: game.name }} (${plural({ count: more }, { one: '# more profile', other: '# more profiles' })})`}
          onClick={onOpenGame}
          sx={{
            flexShrink: 0,
            maxHeight: 28,
            px: 0.75,
            overflow: 'hidden',
            color: 'inherit',
            fontFamily: 'inherit',
            fontSize: 'base.fontSize',
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
  profile,
  lastPlayedAt,
  onOpen,
}: {
  profile: Profile
  lastPlayedAt: string
  onOpen: (ev: MouseEvent) => void
}) {
  return (
    <ButtonBase
      component="div"
      onClick={onOpen}
      sx={{
        display: 'flex',
        flexDirection: 'row',
        flexWrap: 'nowrap',
        alignItems: 'center',
        gap: space.gap,
        flexShrink: 0,
        height: 32,
        px: 1.25,
        minWidth: 0,
        maxWidth: '100%',
        overflow: 'hidden',
        color: 'text.primary',
        textShadow: 'none',
        bgcolor: 'var(--mortar-raised)',
        borderRadius: '6px',
        fontFamily: 'inherit',
        textAlign: 'left',
        '&:hover': { bgcolor: 'var(--mortar-hairline-16)' },
      }}
    >
      <Typography
        noWrap={true}
        title={profile.name}
        sx={{ fontSize: 'base.fontSize', fontWeight: 700, lineHeight: 1.2 }}
      >
        {profile.name}
      </Typography>
      {lastPlayedAt ? (
        <Box
          sx={{
            fontSize: 'small.fontSize',
            lineHeight: 1.2,
            overflow: 'hidden',
            whiteSpace: 'nowrap',
            color: 'text.secondary',
          }}
        >
          <When value={lastPlayedAt} />
        </Box>
      ) : null}
    </ButtonBase>
  )
}

export { ProfileCards }
