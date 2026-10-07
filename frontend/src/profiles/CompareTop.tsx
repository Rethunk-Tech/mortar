import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  IconButton,
  ListItemIcon,
  ListItemText,
  ListSubheader,
  Menu,
  MenuItem,
  Typography,
} from '@mui/material'
import { Check, ChevronDown, Search, Users, X } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { modsLabel } from '../i18n/counts.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { space } from '../theme/density.ts'
import { colorHex } from './appearance.ts'
import { userModCount } from './count.ts'
import { useProfiles } from './store.ts'

function Dot({ profile }: { profile: Profile }) {
  return (
    <Box
      component="span"
      aria-hidden={true}
      sx={{
        width: 10,
        height: 10,
        borderRadius: '50%',
        flexShrink: 0,
        bgcolor: colorHex(profile.color) ?? 'primary.main',
      }}
    />
  )
}

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
  const heading = { fontSize: 11, fontWeight: 700, letterSpacing: '0.08em', lineHeight: '28px' }
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
        {others.length > 0 ? <ListSubheader sx={heading}>{t`YOUR PROFILES`}</ListSubheader> : null}
        {others.map((p) => (
          <MenuItem key={p.id} onClick={choose(() => onPick(p.id))}>
            <ListItemIcon>
              <Dot profile={p} />
            </ListItemIcon>
            <ListItemText>{t`${p.name} · ${modsLabel(userModCount(p))}`}</ListItemText>
            {p.id === profileB?.id ? <Check size={16} aria-hidden={true} /> : null}
          </MenuItem>
        ))}
        <ListSubheader sx={heading}>{t`SOMEONE ELSE'S`}</ListSubheader>
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
  onClose,
  onPick,
  onFriend,
  onHost,
}: {
  profileA: Profile
  profileB: Profile | null
  pending: boolean
  filterOpen: boolean
  onFilter: () => void
  onClose: () => void
  onPick: (id: string) => void
  onFriend: () => void
  onHost: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const gameId = game?.id ?? ''
  return (
    <>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'baseline',
          gap: space.gap,
          px: space.pad,
          pt: space.pad,
          pb: space.pad,
        }}
      >
        <Typography component="h2" sx={{ fontSize: 20, fontWeight: 700 }}>
          {t`Compare`}
        </Typography>
        <Typography color="text.secondary">{game?.name ?? ''}</Typography>
        <Box sx={{ flex: 1 }} />
        <IconButton size="small" aria-label={t`Close`} onClick={onClose} disabled={pending}>
          <X size={18} />
        </IconButton>
      </Box>
      <Box
        sx={{ display: 'flex', alignItems: 'center', gap: space.gap, px: space.pad, pb: space.pad }}
      >
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
          <Dot profile={profileA} />
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
    </>
  )
}
