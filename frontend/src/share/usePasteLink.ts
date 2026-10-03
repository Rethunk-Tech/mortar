import { useLingui } from '@lingui/react/macro'
import { useEffect } from 'react'
import { useLocked } from '../mods/useLocked.ts'
import { dialogOpen } from '../settings/shortcuts.ts'
import { useToasts } from '../toasts/store.ts'
import { classifyPastedLink, pasteTargetIsEditable } from './pasteLink.ts'
import { openImport } from './store.ts'

export function usePasteLink(profileId: string) {
  const { t } = useLingui()
  const locked = useLocked()
  useEffect(() => {
    const onPaste = (e: ClipboardEvent) => {
      const target = e.target instanceof Element ? e.target : null
      if (pasteTargetIsEditable(target) || pasteTargetIsEditable(document.activeElement)) {
        return
      }
      if (dialogOpen()) {
        return
      }
      const text = e.clipboardData?.getData('text')?.trim() ?? ''
      if (!classifyPastedLink(text)) {
        return
      }
      e.preventDefault()
      if (locked) {
        useToasts.getState().push({
          kind: 'warning',
          title: t`Stop the game to change mods.`,
        })
        return
      }
      openImport({ profileId, link: text })
    }
    document.addEventListener('paste', onPaste)
    return () => document.removeEventListener('paste', onPaste)
  }, [locked, profileId, t])
}
