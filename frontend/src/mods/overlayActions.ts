import { msg } from '@lingui/core/macro'
import type {
  Entry,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  RemoveEntries,
  RestoreEntries,
  SetOverlayEnabled,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { overlayLabel } from './overlayRows.ts'
import { openTarget } from './storeView.ts'
import type { OverlayRow } from './virtualRows.ts'

const overlaysOn = (p: Profile | undefined, baseKey: string) =>
  (p?.entries ?? []).filter((e) => e.overlayOf === baseKey && !e.overlayOff)

// Switching an optional file on switches its alternatives off; the toast names them and Undo switches them back on.
function toastSwitchedOff(row: OverlayRow, before: Entry[], after: Profile, profileId: string) {
  const stillOn = new Set(overlaysOn(after, row.baseKey).map((e) => e.key))
  const off = before.filter((e) => e.key !== row.key && !stillOn.has(e.key))
  if (off.length === 0) {
    return
  }
  const names = off.map(overlayLabel).join(', ')
  useToasts.getState().push({
    kind: 'success',
    title: i18n._(msg`Switched on ${row.label}`),
    body: i18n._(msg`It replaces the same files, so this is switched off: ${names}`),
    action: {
      label: i18n._(msg`Undo`),
      profileId,
      run: async () => {
        const now = openTarget()
        for (const e of off) {
          if (now) {
            useProfiles.getState().replace(await SetOverlayEnabled(now.game, now.id, e.key, true))
          }
        }
      },
      live: () => {
        const cur = useProfiles.getState().profiles.find((p) => p.id === profileId)
        return overlaysOn(cur, row.baseKey).some((e) => e.key === row.key)
          ? { disabled: false }
          : { disabled: true, reason: i18n._(msg`This is no longer the latest change.`) }
      },
    },
  })
}

export async function setOverlayEnabled(row: OverlayRow, enabled: boolean) {
  const target = openTarget()
  if (!target) {
    return
  }
  const before = overlaysOn(
    useProfiles.getState().profiles.find((p) => p.id === target.id),
    row.baseKey,
  )
  try {
    const after = await SetOverlayEnabled(target.game, target.id, row.key, enabled)
    useProfiles.getState().replace(after)
    if (enabled) {
      toastSwitchedOff(row, before, after, target.id)
    }
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
