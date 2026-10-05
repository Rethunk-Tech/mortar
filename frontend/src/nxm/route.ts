import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Route } from '../nav/store.ts'
import { openProfileOf } from '../profiles/store.ts'

// The profile a link installs into without asking: the open profile while the link's game screen is showing. Any
// other screen (game select, settings, first run, profiles page) has no such profile and the user is asked.
// loaded is the profiles store: the game whose profiles it holds, its open profile and the profiles.
export function directProfile(
  route: Route,
  linkGame: string,
  loaded: { game: string | undefined; openId: string; profiles: Profile[] },
): Profile | null {
  const { game, openId, profiles } = loaded
  if (route.name !== 'game' || route.game !== linkGame || game !== linkGame) {
    return null
  }
  return openProfileOf({ profiles, openId }) ?? null
}
