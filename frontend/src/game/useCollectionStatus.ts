import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { CollectionStatus } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import { CollectionStatus as loadCollectionStatus } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'
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

export function useCollectionStatus(
  game: string,
  profile: Profile | undefined,
): CollectionLink | null {
  const { data: status } = useLoaded<CollectionStatus | null>(
    profile?.collection ? () => loadCollectionStatus(game, profile.id) : null,
    [game, profile],
    null,
  )
  if (!profile?.collection) {
    return null
  }
  if (status) {
    return status
  }
  return fallbackStatus(profile)
}
