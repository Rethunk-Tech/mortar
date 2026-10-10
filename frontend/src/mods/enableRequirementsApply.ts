import { msg } from '@lingui/core/macro'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetModsEnabled } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { openOverride } from '../profiles/openOverrides.ts'
import { useProfiles } from '../profiles/store.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { reportError } from '../toasts/report.ts'
import { useEnableAsk } from './enableAsk.ts'
import { enableRequirementsDecision, pendingRequired } from './enableRequirements.ts'
import { announceAlso, openTarget } from './storeView.ts'

// What to do with a mod's switched-off requirements, as the open profile sees it: its own override, else the game's
// setting. Go enables by the same resolved value, so the list must not decide by the game's value alone.
export function enableRequirementsMode(): string {
  return openOverride(
    'enableRequirements',
    gamePrefs(useSettings.getState()).enableRequirements || 'always',
  )
}

export function considerEnableRequirements(
  allMods: Mod[],
  enabling: Mod[],
  source: 'toggle' | 'install',
): Promise<void> {
  if (enabling.length === 0) {
    return Promise.resolve()
  }
  const pending = pendingRequired(allMods, enabling)
  const decision = enableRequirementsDecision(enableRequirementsMode(), pending.length)
  if (decision === 'skip' || (decision === 'enable' && source === 'toggle')) {
    return Promise.resolve()
  }
  const dependentName = listNames(enabling.map((m) => m.name).filter(Boolean))
  if (decision === 'ask') {
    useEnableAsk.getState().enqueue({ dependentName, mods: pending })
    return Promise.resolve()
  }
  const target = openTarget()
  if (!target) {
    return Promise.resolve()
  }
  return SetModsEnabled(
    target.game,
    target.id,
    pending.map((m) => ({ key: m.key, id: m.id })),
    true,
  )
    .then((r) => {
      useProfiles.getState().replace(r.profile)
      announceAlso(r.alsoEnabled?.length ? r.alsoEnabled : pending.map((m) => m.name))
    })
    .catch(reportError(i18n._(msg`Could not enable required mods`)))
}
