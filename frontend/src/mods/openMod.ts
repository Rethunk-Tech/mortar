import type { MouseEvent } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useTab } from '../game/tab.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDetail } from './detail.ts'
import { sameId } from './lookup.ts'
import { useMods } from './store.ts'

interface Target {
  id: string
  // key narrows to one entry when the same mod id sits in several.
  key?: string
}

const findMod = ({ id, key }: Target) =>
  useMods.getState().mods.find((m) => sameId(m.id, id) && (!key || m.key === key))

// openMod switches to the Mods tab, filters the list to the open profile's mod and shows it in the details panel, loading
// the list first when the mod is not in it yet.
export async function openMod(target: Target): Promise<void> {
  useTab.getState().setTab('mods')
  let mod = findMod(target)
  if (!mod) {
    await useMods.getState().load()
    mod = findMod(target)
  }
  if (!mod) {
    return
  }
  focusMod(mod)
}

// focusMod filters the Mods list to mod and shows it in the details panel: what a link to a mod lands on.
export function focusMod(mod: Mod): void {
  const mods = useMods.getState()
  mods.setQuery(mods.modsFor, mod.name)
  useDetail.getState().show(mod)
}

// openModHandlers are the click and right-click that open a mod's details, as a Mods row does.
export function openModHandlers(target: Target) {
  const run = () => {
    openMod(target).catch(reportUnexpected)
  }
  return {
    onClick: (e: MouseEvent) => {
      e.stopPropagation()
      run()
    },
    onContextMenu: (e: MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      run()
    },
  }
}
