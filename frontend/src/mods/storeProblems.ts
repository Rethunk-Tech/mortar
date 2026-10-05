import { msg } from '@lingui/core/macro'
import type {
  AssetConflict,
  SettingHint,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/models.ts'
import type { Duplicate } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import {
  DismissAbandonedMod,
  DismissAssetConflict,
  DismissListedRequirement,
  DismissSetting,
  RestoreDismissed,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import {
  SetConfigValue,
  SetModEnabled,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportError } from '../toasts/report.ts'
import { openTarget } from './storeView.ts'

async function dismissAbandonedMod(get: () => { loadProblems: () => Promise<void> }, id: string) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await DismissAbandonedMod(target.game, target.id, id)
  } catch (e) {
    reportError(i18n._(msg`Could not dismiss the warning`))(e)
    return
  }
  await get().loadProblems()
}

async function dismissListedRequirement(
  get: () => { loadProblems: () => Promise<void> },
  id: string,
) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await DismissListedRequirement(target.game, target.id, id)
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
    await DismissSetting(target.game, target.id, setting.id, setting.field)
  } catch (e) {
    reportError(i18n._(msg`Could not dismiss the setting warning`))(e)
    return
  }
  await get().loadProblems()
}

async function setConfigSetting(
  get: () => { loadProblems: () => Promise<void> },
  setting: Pick<SettingHint, 'key' | 'id' | 'field' | 'name'>,
  value: string,
) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await SetConfigValue(target.game, target.id, setting.key, setting.id, setting.field, value)
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
            .replace((await SetModEnabled(target.game, target.id, c.key, dup.id, false)).profile)
        }
      } catch (e) {
        reportError(i18n._(msg`Could not disable the other copy of ${dup.name}`))(e)
      }
      set({ resolving: null })
      await get().load()
    },
    dismissAsset: (conflict: AssetConflict) => dismissAssetConflict(get, conflict),
    restoreDismissed: (token: string) => restoreDismissed(get, token),
    dismissAbandoned: (id: string) => dismissAbandonedMod(get, id),
    dismissListed: (id: string) => dismissListedRequirement(get, id),
    dismissSetting: (setting: SettingHint) => dismissSettingHint(get, setting),
    setConfigValue: (setting: Pick<SettingHint, 'key' | 'id' | 'field' | 'name'>, value: string) =>
      setConfigSetting(get, setting, value),
  }
}
