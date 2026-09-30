import { msg } from '@lingui/core/macro'
import { Browser, Events } from '@wailsio/runtime'
import { useEffect } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useDetail } from './detail.ts'
import { modId } from './lookup.ts'
import { useMods } from './store.ts'

// The state that decides a mod's actions, and the name of the native menu registered for it (modmenu.MenuID).
interface MenuState {
  enabled: boolean
  hasPage: boolean
  removable: boolean
}

const flag = (on: boolean, yes: string, no: string) => (on ? yes : no)

export const menuId = (s: MenuState) =>
  `mod-menu-${flag(s.enabled, 'on', 'off')}-${flag(s.hasPage, 'page', 'nopage')}-${flag(s.removable, 'remove', 'keep')}`

export function useMenuState(mod: Mod, removable: boolean): MenuState {
  const hasPage = useMods((s) => modId(mod) in s.pages)
  return { enabled: mod.enabled, hasPage, removable }
}

// The style that makes a right-click on the element open the native menu of the mod.
export function useContextMenuSx(mod: Mod, removable: boolean) {
  const state = useMenuState(mod, removable)
  const game = useProfiles((s) => s.game?.id ?? '')
  const profile = useProfiles((s) => s.openId)
  return {
    '--custom-contextmenu': menuId(state),
    '--custom-contextmenu-data': JSON.stringify({
      game,
      profile,
      key: mod.key,
      uniqueId: mod.uniqueId,
    }),
  }
}

export const openPage = (url: string) => Browser.OpenURL(url)

// Applies what the native menu asks of the window: open a mod's details or its remove confirmation, refresh after
// a switch, and report a failure.
export function useModMenuEvents() {
  useEffect(() => {
    const mine = (t: { profile: string; key: string; uniqueId: string }) => {
      const open = useProfiles.getState().openId
      return open === t.profile
        ? useMods.getState().mods.find((m) => m.key === t.key && m.uniqueId === t.uniqueId)
        : undefined
    }
    const offs = [
      Events.On('mod:details', (e) => {
        const mod = mine(e.data)
        if (mod) {
          useDetail.getState().show(mod)
          useDetail.getState().setOpen(true)
        }
      }),
      Events.On('mod:remove', (e) => {
        const mod = mine(e.data)
        if (mod) {
          useMods.getState().askRemove(mod)
        }
      }),
      Events.On('mods:changed', (e) => {
        if (useProfiles.getState().openId === e.data.id) {
          useProfiles.getState().replace(e.data)
          useMods.getState().load().catch(reportUnexpected)
        }
      }),
      Events.On('mod:failed', (e) => {
        useToasts
          .getState()
          .push({ kind: 'error', title: i18n._(msg`Could not run that action`), body: e.data })
      }),
    ]
    return () => {
      for (const off of offs) {
        off()
      }
    }
  }, [])
}
