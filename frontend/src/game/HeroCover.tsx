import { Box } from '@mui/material'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { Covers } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { firstCoverSrc } from './cover.ts'

// The first of the profile's covers that loads (picked, Nexus, Steam), else a solid tone.
export function HeroCover({ game, profile }: { game: string; profile: Profile }) {
  // Kept with the `updated` it was read for: the profile's cover and mods change with it.
  const [failed, setFailed] = useState<{ stamp: string; urls: string[] }>({ stamp: '', urls: [] })
  const updated = String(profile.updated)
  const { data: covers } = useLoaded<{ stamp: string; list: string[] } | null>(
    () =>
      Covers(game, profile.id).then(
        (list) => ({ stamp: updated, list: list ?? [] }),
        (e: unknown) => {
          reportUnexpected(e)
          return { stamp: updated, list: [] }
        },
      ),
    [game, profile.id, updated],
    null,
  )
  if (covers === null) {
    return <Box sx={{ width: '100%', height: '100%', bgcolor: 'var(--mortar-panel-solid)' }} />
  }
  const skip = failed.stamp === covers.stamp ? failed.urls : []
  const src = firstCoverSrc(covers.list, skip)
  return src ? (
    <Box
      component="img"
      src={src}
      alt=""
      onError={() => setFailed({ stamp: covers.stamp, urls: [...skip, src] })}
      sx={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
    />
  ) : (
    <Box sx={{ width: '100%', height: '100%', bgcolor: 'var(--mortar-panel-solid)' }} />
  )
}
