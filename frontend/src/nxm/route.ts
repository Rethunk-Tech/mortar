import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Route } from '../nav/store.ts'
import { openProfileOf } from '../profiles/store.ts'

// The profile a link installs into without asking: the open profile while the link's game screen is showing. Any
// other screen (game select, settings, first run, profiles page) has no such profile and the user is asked.
export function directProfile(
  route: Route,
  linkGame: string,
  gameId: string | undefined,
  openId: string,
  profiles: Profile[],
): Profile | null {
  if (route.name !== 'game' || route.game !== linkGame || gameId !== linkGame) {
    return null
  }
  return openProfileOf({ profiles, openId }) ?? null
}
