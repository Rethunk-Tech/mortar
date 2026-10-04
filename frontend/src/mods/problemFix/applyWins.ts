import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetWinner } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useMods } from '../store.ts'

// applyWins makes winnerKey load after every other pack in packIds (skipping index skip), so its tied edits apply last.
export async function applyWins(
  winnerKey: string,
  packIds: string[],
  skip: number,
  on: boolean,
): Promise<void> {
  const { game, openId, replace } = useProfiles.getState()
  if (!(game && openId)) {
    return
  }
  let profile: Profile | undefined
  for (const [i, id] of packIds.entries()) {
    if (i !== skip && id) {
      profile = await SetWinner(game.id, openId, winnerKey, id, on)
    }
  }
  if (profile) {
    replace(profile)
    await useMods.getState().loadProblems()
  }
}
