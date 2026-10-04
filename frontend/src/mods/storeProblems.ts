import { msg } from '@lingui/core/macro'
import type {
  AssetConflict,
  Duplicate,
  SettingHint,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import {
  DismissAbandonedMod,
  DismissAssetConflict,
  DismissListedRequirement,
  DismissSetting,
  RestoreDismissed,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import {
  SetConfigValue,
  SetModEnabled,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportError } from '../toasts/report.ts'
import { openTarget } from './storeView.ts'

async function dismissAbandonedMod(
  get: () => { loadProblems: () => Promise<void> },
  uniqueId: string,
) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await DismissAbandonedMod(target.game, target.id, uniqueId)
  } catch (e) {
    reportError(i18n._(msg`Could not dismiss the warning`))(e)
    return
  }
  await get().loadProblems()
}

async function dismissListedRequirement(
  get: () => { loadProblems: () => Promise<void> },
  uniqueId: string,
) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await DismissListedRequirement(target.game, target.id, uniqueId)
  } catch (e) {
    reportError(i18n._(msg`Could not dismiss the warning`))(e)
    return
  }
  await get().loadProblems()
}

async function dismissSettingHint(
  get: () => { loadProblems: () => Promise<void> },
  setting: SettingHint,
) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await DismissSetting(target.game, target.id, setting.uniqueId, setting.field)
  } catch (e) {
    reportError(i18n._(msg`Could not dismiss the setting warning`))(e)
    return
  }
  await get().loadProblems()
}

async function setConfigSetting(
  get: () => { loadProblems: () => Promise<void> },
  setting: Pick<SettingHint, 'key' | 'uniqueId' | 'field' | 'name'>,
  value: string,
) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await SetConfigValue(
      target.game,
      target.id,
      setting.key,
      setting.uniqueId,
      setting.field,
      value,
    )
  } catch (e) {
    reportError(i18n._(msg`Could not set ${setting.field} for ${setting.name}`))(e)
    return
  }
  await get().loadProblems()
}

async function dismissAssetConflict(
  get: () => { loadProblems: () => Promise<void> },
  conflict: AssetConflict,
) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await DismissAssetConflict(target.game, target.id, conflict.kind, conflict.target)
  } catch (e) {
    reportError(i18n._(msg`Could not dismiss the overlap`))(e)
    return
  }
  await get().loadProblems()
}

async function restoreDismissed(get: () => { loadProblems: () => Promise<void> }, token: string) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await RestoreDismissed(target.game, target.id, token)
  } catch (e) {
    reportError(i18n._(msg`Could not restore the warning`))(e)
    return
  }
  await get().loadProblems()
}

export function problemActions(
  set: (p: { resolving?: Duplicate | null }) => void,
  get: () => { load: () => Promise<void>; loadProblems: () => Promise<void> },
) {
  return {
    resolve: (resolving: Duplicate | null) => set({ resolving }),
    keepCopy: async (dup: Duplicate, keepKey: string) => {
      const target = openTarget()
      if (!target) {
        return
      }
      try {
        for (const c of (dup.copies ?? []).filter((x) => x.key !== keepKey)) {
          useProfiles
            .getState()
            .replace(
              (await SetModEnabled(target.game, target.id, c.key, dup.uniqueId, false)).profile,
            )
        }
      } catch (e) {
        reportError(i18n._(msg`Could not switch off the other copy of ${dup.name}`))(e)
      }
      set({ resolving: null })
      await get().load()
    },
    dismissAsset: (conflict: AssetConflict) => dismissAssetConflict(get, conflict),
    restoreDismissed: (token: string) => restoreDismissed(get, token),
    dismissAbandoned: (uniqueId: string) => dismissAbandonedMod(get, uniqueId),
    dismissListed: (uniqueId: string) => dismissListedRequirement(get, uniqueId),
    dismissSetting: (setting: SettingHint) => dismissSettingHint(get, setting),
    setConfigValue: (
      setting: Pick<SettingHint, 'key' | 'uniqueId' | 'field' | 'name'>,
      value: string,
    ) => setConfigSetting(get, setting, value),
  }
}
