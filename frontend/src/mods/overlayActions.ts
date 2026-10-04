import { msg } from '@lingui/core/macro'
import {
  RemoveEntries,
  RestoreEntries,
  SetOverlayEnabled,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { openTarget } from './storeView.ts'
import type { OverlayRow } from './virtualRows.ts'

export async function setOverlayEnabled(row: OverlayRow, enabled: boolean) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    useProfiles
      .getState()
      .replace(await SetOverlayEnabled(target.game, target.id, row.key, enabled))
  } catch (e) {
    reportError(
      enabled
        ? i18n._(msg`Could not switch on ${row.label}`)
        : i18n._(msg`Could not switch off ${row.label}`),
    )(e)
  }
}

export async function removeOverlay(row: OverlayRow) {
  const target = openTarget()
  if (!target) {
    return
  }
  const profile = useProfiles.getState().profiles.find((p) => p.id === target.id)
  const entry = (profile?.entries ?? []).find((e) => e.key === row.key)
  try {
    useProfiles.getState().replace(await RemoveEntries(target.game, target.id, [row.key]))
  } catch (e) {
    reportError(i18n._(msg`Could not remove ${row.label}`))(e)
    return
  }
  if (!entry) {
    return
  }
  useToasts.getState().push({
    kind: 'success',
    title: i18n._(msg`Removed ${row.label}`),
    action: {
      label: i18n._(msg`Undo`),
      profileId: target.id,
      run: async () => {
        const now = openTarget()
        if (now) {
          useProfiles.getState().replace(await RestoreEntries(now.game, now.id, [entry]))
        }
      },
      live: () => {
        const cur = useProfiles.getState().profiles.find((p) => p.id === target.id)
        const entries = cur?.entries ?? []
        if (entries.some((e) => e.key === row.key)) {
          return { disabled: true, reason: i18n._(msg`This is no longer the latest change.`) }
        }
        return entries.some((e) => e.key === row.baseKey)
          ? { disabled: false }
          : { disabled: true, reason: i18n._(msg`Its main file is no longer in this profile.`) }
      },
    },
  })
}
