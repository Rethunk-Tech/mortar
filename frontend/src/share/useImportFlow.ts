import { msg, plural } from '@lingui/core/macro'
import { useCallback, useRef, useState } from 'react'
import { RegisterLinks } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/service.ts'
import { SetLastGame } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import type {
  Preview,
  Result,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import {
  Discard,
  Import,
  PickFile,
  PreviewFile,
  PreviewLink,
  ReadClipboard,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useQueue } from '../queue/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { beginWork } from '../toasts/usePending.ts'
import { type ShownPreview, shownPreview } from './logic.ts'
import { importAfterSignIn, useImportDialog } from './store.ts'

function shouldOpenQueueAfterImport(queued: number): boolean {
  return queued > 0
}

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

// Tells what an import did: the downloads it queued, and the name a new profile took when the shared one was taken.
function announce(result: Result, intoOpen: boolean, sharedName: string | undefined) {
  const { name } = result.profile
  const { queued } = result
  const renamed = !intoOpen && sharedName !== undefined && name !== sharedName
  const body = [
    queued > 0
      ? plural(queued, { one: '# download queued', other: '# downloads queued' })
      : i18n._(msg`Nothing to download`),
    renamed ? i18n._(msg`A profile with that name already exists, so this one is "${name}"`) : '',
  ]
    .filter(Boolean)
    .join('. ')
  useToasts.getState().push({
    kind: 'info',
    title: queued > 0 ? i18n._(msg`Importing into ${name}`) : i18n._(msg`Imported ${name}`),
    body,
  })
  if (shouldOpenQueueAfterImport(queued)) {
    useQueue.getState().setOpen(true)
  }
}

async function afterImport(
  game: string,
  intoOpen: boolean,
  result: Result,
  previewName: string | undefined,
) {
  try {
    await showImported(game, intoOpen, result.profile.id)
    announce(result, intoOpen, previewName)
  } catch (e) {
    useToasts.getState().push({
      kind: 'error',
      title: i18n._(msg`Imported, but could not show the profile`),
      body: errorMessage(e),
    })
  }
}

export type Tab = 'link' | 'file'

// The import dialog's state: what was typed or picked, the preview it produced and the mods unticked.
export function useImportFlow(game: string, profileId: string, close: () => void) {
  const [tab, setTab] = useState<Tab>('link')
  const [text, setText] = useState('')
  const [path, setPath] = useState('')
  const [preview, setPreview] = useState<ShownPreview | null>(null)
  const [excluded, setExcluded] = useState<ReadonlySet<string>>(new Set())
  const [error, setError] = useState('')
  const [previewBusy, setPreviewBusy] = useState(false)
  const [importBusy, setImportBusy] = useState(false)
  // Each preview or reset takes a new number; a result for an older one arrived after it was overtaken.
  const latest = useRef(0)
  const importing = useRef(false)
  const show = useCallback(async (pending: Promise<Preview>) => {
    const n = latest.current + 1
    latest.current = n
    setPreviewBusy(true)
    setError('')
    try {
      const shown = shownPreview(await pending)
      if (n === latest.current) {
        setPreview(shown)
        setExcluded(new Set())
      }
    } catch (e) {
      if (n === latest.current) {
        setPreview(null)
        setError(errorMessage(e))
      }
    } finally {
      if (n === latest.current) {
        setPreviewBusy(false)
      }
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
    latest.current += 1
    setPreviewBusy(false)
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

  const run = async (intoOpen: boolean) => {
    if (!beginWork(importing)) {
      return
    }
    setImportBusy(true)
    const opened = useImportDialog.getState().request?.run
    try {
      const result = await Import(game, preview?.session ?? '', intoOpen ? profileId : '', [
        ...excluded,
      ])
      await afterImport(game, intoOpen, result, preview?.name)
      if (useImportDialog.getState().request?.run === opened) {
        close()
      }
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      importing.current = false
      setImportBusy(false)
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
    busy: previewBusy || importBusy,
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
export { shouldOpenQueueAfterImport }
