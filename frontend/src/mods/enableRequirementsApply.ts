import { msg } from '@lingui/core/macro'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { SetModsEnabled } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { reportError } from '../toasts/report.ts'
import { useEnableAsk } from './enableAsk.ts'
import { enableRequirementsDecision, pendingRequired } from './enableRequirements.ts'
import { announceAlso, open } from './storeView.ts'

export function considerEnableRequirements(
  allMods: Mod[],
  enabling: Mod[],
  source: 'toggle' | 'install',
): Promise<void> {
  if (enabling.length === 0) {
    return Promise.resolve()
  }
  const pending = pendingRequired(allMods, enabling)
  const decision = enableRequirementsDecision(
    gamePrefs(useSettings.getState()).enableRequirements || 'always',
    pending.length,
  )
  if (decision === 'skip' || (decision === 'enable' && source === 'toggle')) {
    return Promise.resolve()
  }
  const dependentName =
    enabling
      .map((m) => m.name)
      .filter(Boolean)
      .join(', ') ||
    enabling[0]?.name ||
    ''
  if (decision === 'ask') {
    useEnableAsk.getState().enqueue({ dependentName, mods: pending })
    return Promise.resolve()
  }
  const target = open()
  if (!target) {
    return Promise.resolve()
  }
  return SetModsEnabled(
    target.game,
    target.id,
    pending.map((m) => ({ key: m.key, uniqueId: m.uniqueId })),
    true,
  )
    .then((r) => {
      useProfiles.getState().replace(r.profile)
      announceAlso(r.alsoEnabled?.length ? r.alsoEnabled : pending.map((m) => m.name))
    })
    .catch(reportError(i18n._(msg`Could not enable required mods`)))
}
