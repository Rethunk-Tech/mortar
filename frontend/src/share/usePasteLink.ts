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
    // A link dragged from a browser arrives as text, not as a file, so the native file drop never sees it.
    const isLinkDrag = (e: DragEvent) => {
      const types = e.dataTransfer?.types ?? []
      return (
        !types.includes('Files') &&
        (types.includes('text/uri-list') || types.includes('text/plain'))
      )
    }
    const onDragOver = (e: DragEvent) => {
      if (isLinkDrag(e)) {
        e.preventDefault()
      }
    }
    const onDrop = (e: DragEvent) => {
      if (!isLinkDrag(e) || dialogOpen()) {
        return
      }
      const raw =
        e.dataTransfer?.getData('text/uri-list') || e.dataTransfer?.getData('text/plain') || ''
      const text =
        raw
          .split('\n')
          .find((line) => line.trim() !== '' && !line.startsWith('#'))
          ?.trim() ?? ''
      if (!classifyPastedLink(text)) {
        return
      }
      e.preventDefault()
      if (locked) {
        useToasts.getState().push({ kind: 'warning', title: t`Stop the game to change mods.` })
        return
      }
      openImport({ profileId, link: text })
    }
    document.addEventListener('paste', onPaste)
    document.addEventListener('dragover', onDragOver)
    document.addEventListener('drop', onDrop)
    return () => {
      document.removeEventListener('paste', onPaste)
      document.removeEventListener('dragover', onDragOver)
      document.removeEventListener('drop', onDrop)
    }
  }, [locked, profileId, t])
}
