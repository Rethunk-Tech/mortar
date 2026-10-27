import { Box } from '@mui/material'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ProfileMark } from './ProfileMark.tsx'

/** A profile's mark at icon size where it has a colour or icon, else a plain dot, as menus list profiles. */
export function ProfileDot({
  profile,
  size = 16,
}: {
  profile: Profile | undefined
  size?: number
}) {
  if (profile && (profile.color || profile.icon)) {
    return <ProfileMark profile={profile} size={size} />
  }
  return (
    <Box
      component="span"
      aria-hidden={true}
      sx={{ width: 10, height: 10, flexShrink: 0, borderRadius: '50%', bgcolor: 'primary.main' }}
    />
  )
}
