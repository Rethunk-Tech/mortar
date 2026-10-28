import type { KeyboardEvent } from 'react'

/** onKeyDown for a field that commits on blur: Enter leaves the field, which commits it. */
export function blurOnEnter(e: KeyboardEvent) {
  if (e.key === 'Enter' && e.target instanceof HTMLInputElement) {
    e.target.blur()
  }
}
