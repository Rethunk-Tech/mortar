import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { useCallback, useState } from 'react'
import { RegisterLinks } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/service.ts'
import { SetLastGame } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import type { Preview } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import {
  Discard,
  Import,
  PickFile,
  PreviewFile,
  PreviewLink,
  ReadClipboard,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/service.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useQueue } from '../queue/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { type ShownPreview, shownPreview } from './logic.ts'
import { importAfterSignIn } from './store.ts'

// Shows what an import filled: the open profile refreshed, or the new one opened on its game's page.
// A new profile registers mortar:// and .mortar (idempotent): an import can arrive before first run finished,
// including via the Nexus sign-in detour, which leaves the setup route behind.
async function showImported(game: string, intoOpen: boolean, id: string) {
  if (intoOpen) {
    await useProfiles.getState().refresh()
  } else {
    await useProfiles.getState().load(game)
    useProfiles.getState().open(id)
    await SetLastGame(game)
    useNav.getState().openGame('stardew')
    RegisterLinks().catch(reportUnexpected)
  }
}

export type Tab = 'link' | 'file'

// The import dialog's state: what was typed or picked, the preview it produced and the mods unticked.
export function useImportFlow(game: string, profileId: string, close: () => void) {
  const { t } = useLingui()
  const [tab, setTab] = useState<Tab>('link')
  const [text, setText] = useState('')
  const [path, setPath] = useState('')
  const [preview, setPreview] = useState<ShownPreview | null>(null)
  const [excluded, setExcluded] = useState<ReadonlySet<string>>(new Set())
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const show = useCallback(async (pending: Promise<Preview>) => {
    setBusy(true)
    setError('')
    try {
      setPreview(shownPreview(await pending))
      setExcluded(new Set())
    } catch (e) {
      setPreview(null)
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }, [])

  const previewLink = useCallback(
    (value: string) => show(PreviewLink(game, value, profileId)),
    [game, profileId, show],
  )
  const previewFile = useCallback(
    (file: string) => {
      setPath(file)
      return show(PreviewFile(game, file, profileId))
    },
    [game, profileId, show],
  )

  // The clipboard is read only here, when the user presses Paste.
  const paste = async () => {
    const clip = await ReadClipboard()
    setText(clip)
    await previewLink(clip)
  }
  const pick = async () => {
    const file = await PickFile()
    if (file) {
      await previewFile(file)
    }
  }
  const reset = () => {
    Discard().catch(reportUnexpected)
    setPreview(null)
    setExcluded(new Set())
    setError('')
    setText('')
    setPath('')
  }
  const toggle = (key: string) =>
    setExcluded((prev) => {
      const next = new Set(prev)
      if (!next.delete(key)) {
        next.add(key)
      }
      return next
    })

  // Creates the profile (or fills the open one) and queues the downloads; this is the only place anything starts.
  const run = async (intoOpen: boolean) => {
    setBusy(true)
    try {
      const target = intoOpen ? profileId : ''
      const result = await Import(game, preview?.session ?? '', target, [...excluded])
      await showImported(game, intoOpen, result.profile.id)
      const { name } = result.profile
      const { queued } = result
      const renamed = !intoOpen && preview !== null && name !== preview.name
      const body = [
        queued > 0
          ? plural(queued, { one: '# download queued', other: '# downloads queued' })
          : t`Nothing to download`,
        renamed ? t`A profile with that name already exists, so this one is "${name}"` : '',
      ]
        .filter(Boolean)
        .join('. ')
      useToasts.getState().push({
        kind: 'info',
        title: queued > 0 ? t`Importing into ${name}` : t`Imported ${name}`,
        body,
      })
      if (queued > 0) {
        useQueue.getState().setOpen(true)
      }
      close()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  // The dialog closes for Nexus settings and comes back with this link or file after sign-in.
  const signIn = () => {
    close()
    importAfterSignIn({ profileId, ...(tab === 'file' ? { file: path } : { link: text }) })
  }

  return {
    tab,
    setTab,
    text,
    setText,
    path,
    preview,
    excluded,
    error,
    busy,
    previewLink,
    previewFile,
    paste,
    pick,
    reset,
    dismiss: close,
    toggle,
    run,
    signIn,
  }
}

export type ImportFlow = ReturnType<typeof useImportFlow>
