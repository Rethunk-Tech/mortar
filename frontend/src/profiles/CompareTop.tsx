import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Menu, Typography } from '@mui/material'
import { ChevronDown, Search, Users } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { modsLabel } from '../i18n/counts.ts'
import { MenuAction, MenuCheck } from '../shell/MenuAction.tsx'
import { MenuHeading } from '../shell/TitleMenu.tsx'
import { space } from '../theme/density.ts'
import { userModCount } from './count.ts'
import { ProfileDot } from './ProfileDot.tsx'
import { useProfiles } from './store.ts'

function OtherPicker({
  profileA,
  profileB,
  gameId,
  disabled,
  onPick,
  onFriend,
  onHost,
}: {
  profileA: Profile
  profileB: Profile | null
  gameId: string
  disabled: boolean
  onPick: (id: string) => void
  onFriend: () => void
  onHost: () => void
}) {
  const { t } = useLingui()
  const listed = useProfiles((s) => s.profiles)
  const [menu, setMenu] = useState<HTMLElement | null>(null)
  const others = listed.filter((p) => p.id !== profileA.id)
  const choose = (action: () => void) => () => {
    setMenu(null)
    action()
  }
  return (
    <>
      <Button
        color="inherit"
        variant="outlined"
        disabled={disabled}
        aria-haspopup="menu"
        onClick={(e) => setMenu(e.currentTarget)}
        endIcon={<ChevronDown size={14} />}
        sx={{ height: space.control, whiteSpace: 'nowrap' }}
      >
        {profileB?.name ?? t`Choose…`}
      </Button>
      <Menu anchorEl={menu} open={menu !== null} onClose={() => setMenu(null)}>
        {others.length > 0 ? <MenuHeading>{t`Your profiles`}</MenuHeading> : null}
        {others.map((p) => (
          <MenuCheck
            key={p.id}
            radio={true}
            checked={p.id === profileB?.id}
            icon={<ProfileDot profile={p} />}
            label={`${p.name} · ${modsLabel(userModCount(p))}`}
            onClick={choose(() => onPick(p.id))}
          />
        ))}
        <MenuHeading>{t`Someone else's`}</MenuHeading>
        <MenuAction
          icon={<Users size={16} />}
          label={t`A friend's link or file…`}
          onClick={choose(onFriend)}
        />
        {gameId === 'stardew' ? (
          <MenuAction
            icon={<Users size={16} />}
            label={t`A multiplayer host's list…`}
            onClick={choose(onHost)}
          />
        ) : null}
      </Menu>
    </>
  )
}

export function CompareTop({
  profileA,
  profileB,
  pending,
  filterOpen,
  onFilter,
  onPick,
  onFriend,
  onHost,
}: {
  profileA: Profile
  profileB: Profile | null
  pending: boolean
  filterOpen: boolean
  onFilter: () => void
  onPick: (id: string) => void
  onFriend: () => void
  onHost: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const gameId = game?.id ?? ''
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, px: 3, py: 2 }}>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: space.gap,
          height: space.control,
          px: space.pad,
          borderRadius: '8px',
          bgcolor: 'var(--mortar-hairline-12)',
        }}
      >
        <ProfileDot profile={profileA} />
        <b>{profileA.name}</b>
      </Box>
      <Typography color="text.secondary">{t`with`}</Typography>
      <OtherPicker
        profileA={profileA}
        profileB={profileB}
        gameId={gameId}
        disabled={pending}
        onPick={onPick}
        onFriend={onFriend}
        onHost={onHost}
      />
      <Box sx={{ flex: 1 }} />
      {profileB ? (
        <IconButton
          aria-label={t`Filter mods`}
          aria-pressed={filterOpen}
          onClick={() => onFilter()}
          sx={{
            width: 36,
            height: space.control,
            borderRadius: '8px',
            border: 1,
            borderColor: 'divider',
          }}
        >
          <Search size={16} />
        </IconButton>
      ) : null}
    </Box>
  )
}
