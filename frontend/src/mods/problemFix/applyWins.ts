import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetWinner } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { sameId } from '../lookup.ts'
import { useMods } from '../store.ts'

// applyWins makes pack winnerId of download winnerKey load after every other pack in packIds, so its tied edits
// apply last.
export async function applyWins(
  winnerKey: string,
  winnerId: string,
  packIds: string[],
  on: boolean,
): Promise<void> {
  const { game, openId, replace } = useProfiles.getState()
  if (!(game && openId)) {
    return
  }
  let profile: Profile | undefined
  for (const id of packIds) {
    if (id && !sameId(id, winnerId)) {
      profile = await SetWinner(game.id, openId, winnerKey, winnerId, id, on)
    }
  }
  if (profile) {
    replace(profile)
    await useMods.getState().loadProblems()
  }
}
