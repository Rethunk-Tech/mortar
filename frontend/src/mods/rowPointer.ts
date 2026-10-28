import type { MouseEvent } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { openModPage } from './menu.ts'
import { MIDDLE, opensPage } from './pageClick.ts'
import { useSelection } from './selection.ts'

/** A mod card's or row's pointer handlers: a click selects and opens the details, the middle button opens the page,
 * and Shift or middle presses keep the browser from selecting text or autoscrolling. */
export function rowPointerHandlers(
  mod: Mod,
  id: string,
  orderedRef: { readonly current: readonly string[] },
  openDetail: (mod: Mod) => void,
) {
  return {
    onMouseDown: (e: MouseEvent) => {
      if (e.shiftKey || e.button === MIDDLE) {
        e.preventDefault()
      }
    },
    onAuxClick: (e: MouseEvent) => {
      if (opensPage(e, false)) {
        openModPage(mod)
      }
    },
    onClick: (e: MouseEvent) => {
      useSelection.getState().click(orderedRef.current, id, e)
      openDetail(mod)
    },
  }
}
