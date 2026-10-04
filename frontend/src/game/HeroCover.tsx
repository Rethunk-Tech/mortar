import { Box } from '@mui/material'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { Covers } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { firstCoverSrc } from './cover.ts'

// The first of the profile's covers that loads (picked, Nexus, Steam), else a solid tone.
export function HeroCover({ game, profile }: { game: string; profile: Profile }) {
  // Kept with the `updated` it was read for: the profile's cover and mods change with it.
  const [covers, setCovers] = useState<{ stamp: string; list: string[] } | null>(null)
  const [failed, setFailed] = useState<{ stamp: string; urls: string[] }>({ stamp: '', urls: [] })
  const updated = String(profile.updated)
  useEffect(() => {
    let live = true
    Covers(game, profile.id)
      .then((list) => {
        if (live) {
          setCovers({ stamp: updated, list: list ?? [] })
        }
      })
      .catch((e: unknown) => {
        reportUnexpected(e)
        if (live) {
          setCovers({ stamp: updated, list: [] })
        }
      })
    return () => {
      live = false
    }
  }, [game, profile.id, updated])
  if (covers === null) {
    return <Box sx={{ width: '100%', height: '100%', bgcolor: 'rgb(44,44,54)' }} />
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
    <Box sx={{ width: '100%', height: '100%', bgcolor: 'rgb(44,44,54)' }} />
  )
}
