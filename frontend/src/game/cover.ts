import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ClearCover,
  SetCover,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'

/** `undefined` keeps the saved cover; `null` clears to automatic; a string is a picked path. */
export type StagedCover = string | null | undefined

export function hasPickedCover(savedCover: string | undefined, staged: StagedCover): boolean {
  if (staged === undefined) {
    return Boolean(savedCover)
  }
  return staged !== null
}

export function firstCoverSrc(list: string[], skip: readonly string[]): string | undefined {
  return list.find((c) => !skip.includes(c))
}

export async function applyStagedCover(
  game: string,
  profileId: string,
  staged: StagedCover,
): Promise<Profile | undefined> {
  if (staged === undefined) {
    return undefined
  }
  if (staged === null) {
    return await ClearCover(game, profileId)
  }
  return await SetCover(game, profileId, staged)
}
