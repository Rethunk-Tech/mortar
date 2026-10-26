import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  SetLaunchOptions,
  SetLaunchSettings,
  SetTestCopies,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'

// The profile store's setters for how a profile starts; each saves through the backend and swaps in its reply.

export async function applyLaunchOptions(
  get: () => { game: GameInfo | null; replace: (p: Profile) => void },
  id: string,
  options: string,
) {
  const { game } = get()
  if (game) {
    get().replace(await SetLaunchOptions(game.id, id, options))
  }
}

export async function applyTestCopies(
  get: () => { game: GameInfo | null; replace: (p: Profile) => void },
  id: string,
  n: number,
) {
  const { game } = get()
  if (game) {
    get().replace(await SetTestCopies(game.id, id, n))
  }
}

export async function applyLaunchSettings(
  get: () => { game: GameInfo | null; replace: (p: Profile) => void },
  id: string,
  prefix: string,
  env: string,
) {
  const { game } = get()
  if (game) {
    get().replace(await SetLaunchSettings(game.id, id, prefix, env))
  }
}
