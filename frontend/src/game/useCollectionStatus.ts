import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { CollectionStatus } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import { CollectionStatus as loadCollectionStatus } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/service.ts'
import type { CollectionLink } from './collectionHeader.ts'

function fallbackStatus(profile: Profile): CollectionLink | null {
  if (!profile.collection) {
    return null
  }
  return {
    linked: true,
    name: profile.collection.name,
    revision: profile.collection.revision,
    latest: 0,
    url: `https://www.nexusmods.com/games/${profile.collection.domain}/collections/${profile.collection.slug}`,
  }
}

export function useCollectionStatus(game: string, profile: Profile): CollectionLink | null {
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
        if (!cancelled) {
          setStatus(fallbackStatus(profile))
        }
      })
    return () => {
      cancelled = true
    }
  }, [game, profile])
  if (!profile.collection) {
    return null
  }
  if (status) {
    return status
  }
  return fallbackStatus(profile)
}
