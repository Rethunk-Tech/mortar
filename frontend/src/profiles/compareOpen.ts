import type { MouseEvent } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { CompareSide } from './compare.ts'
import { openModInProfile } from './findMod.ts'

// compareOpen is the click and right-click that open a compared mod's details in the profile holding it.
export function compareOpen(profile: Profile, side: CompareSide) {
  const run = (e: MouseEvent) => {
    e.preventDefault()
    openModInProfile({ profileId: profile.id, key: side.key, id: side.id })
  }
  return { onClick: run, onContextMenu: run }
}
