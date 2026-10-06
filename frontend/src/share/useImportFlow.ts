import { msg, plural } from '@lingui/core/macro'
import { useCallback, useRef, useState } from 'react'
import type { ProfilePreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'
import { RegisterLinks } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/service.ts'
import { SetLastGame } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import type {
  Preview,
  Result,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import {
  Discard,
  Import,
  PickFile,
  PreviewCollectionUpdate,
  PreviewExternal,
  PreviewFile,
  PreviewLink,
  ReadClipboard,
  Replace,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useQueue } from '../queue/store.ts'
import { errorMessage, reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { beginWork } from '../toasts/usePending.ts'
import { trackImport } from './importCompletion.ts'
import { type ShownPreview, shownPreview } from './logic.ts'
import { promptSaveImported } from './savePrompt.ts'
import { type ImportOptions, importAfterSignIn, useImportDialog } from './store.ts'

function shouldOpenQueueAfterImport(queued: number): boolean {
  return queued > 0
}

function signInOptions(options: {
  profileId: string
  external: ProfilePreview | undefined
  tab: Tab
  path: string
  text: string
}): ImportOptions {
  const { profileId, external, tab, path, text } = options
  if (external) {
    return { profileId, external }
  }
  return tab === 'file' ? { profileId, file: path } : { profileId, link: text }
}

function toggleExcluded(previous: ReadonlySet<string>, key: string): ReadonlySet<string> {
  const next = new Set(previous)
  if (!next.delete(key)) {
    next.add(key)
  }
  return next
}

interface PreviewState {
  latest: { current: number }
  setPreviewBusy: (busy: boolean) => void
  setError: (error: string) => void
  setPreview: (preview: ShownPreview | null) => void
  setExcluded: (excluded: ReadonlySet<string>) => void
}

async function showPreview(pending: Promise<Preview>, state: PreviewState) {
  const { latest, setPreviewBusy, setError, setPreview, setExcluded } = state
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
    useNav.getState().openGame(game)
    RegisterLinks().catch(reportUnexpected)
  }
}

function announceCollection(applied: Result['collection']) {
  if (!applied) {
    return
  }
  useToasts.getState().push({
    kind: applied.error ? 'warning' : 'success',
    title: i18n._(
      msg`${plural(applied.fomodMods, {
        one: `Applied the curator's choices for # mod and ${plural(applied.configs, { one: '# config file', other: '# config files' })}`,
        other: `Applied the curator's choices for # mods and ${plural(applied.configs, { one: '# config file', other: '# config files' })}`,
      })}`,
    ),
    ...(applied.error ? { body: applied.error } : {}),
  })
}

// Tells what an import did: the downloads it queued, and the name a new profile took when the shared one was taken.
function announce(
  result: Result,
  intoOpen: boolean,
  notice: { game: string; name: string | undefined; settings: number },
) {
  const { game, name: sharedName, settings: pendingSettings } = notice
  const { name } = result.profile
  const { queued } = result
  trackImport(
    result,
    (batchId) => useQueue.getState().openHistory(batchId),
    pendingSettings,
    useQueue.getState().state.items,
  )
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
    changes: [result.profile.lastChange ?? ''],
    ...(intoOpen
      ? {}
      : {
          action: {
            label: i18n._(msg`Save as template`),
            run: () =>
              promptSaveImported({ game, profileId: result.profile.id, name: sharedName ?? name }),
          },
        }),
  })
  announceCollection(result.collection)
  if (shouldOpenQueueAfterImport(queued)) {
    useQueue.getState().setOpen(true)
  }
}

async function afterImport(
  game: string,
  intoOpen: boolean,
  result: Result,
  notice: { name: string | undefined; settings: number },
) {
  try {
    await showImported(game, intoOpen, result.profile.id)
    announce(result, intoOpen, { game, ...notice })
  } catch (e) {
    toastError(i18n._(msg`Imported, but could not show the profile`), e)
  }
}

export type Tab = 'link' | 'file'

// The import dialog's state: what was typed or picked, the preview it produced and the mods unticked.
export function useImportFlow(
  game: string,
  profileId: string,
  close: () => void,
  external: ProfilePreview | undefined,
) {
  const [tab, setTab] = useState<Tab>('link')
  const [text, setText] = useState('')
  const [path, setPath] = useState('')
  const [preview, setPreview] = useState<ShownPreview | null>(null)
  const [excluded, setExcluded] = useState<ReadonlySet<string>>(new Set())
  const [error, setError] = useState('')
  const [previewBusy, setPreviewBusy] = useState(false)
  const [importBusy, setImportBusy] = useState(false)
  const latest = useRef(0)
  const importing = useRef(false)
  const show = useCallback(
    (pending: Promise<Preview>) =>
      showPreview(pending, { latest, setPreviewBusy, setError, setPreview, setExcluded }),
    [],
  )
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
  const previewCollectionUpdate = useCallback(
    () => show(PreviewCollectionUpdate(game, profileId)),
    [game, profileId, show],
  )
  const previewExternal = useCallback(
    (value: ProfilePreview) => show(PreviewExternal(game, value, profileId)),
    [game, profileId, show],
  )
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
  const toggle = (key: string) => setExcluded((prev) => toggleExcluded(prev, key))
  const run = async (intoOpen: boolean, replace = false) => {
    if (!beginWork(importing)) {
      return
    }
    setImportBusy(true)
    const opened = useImportDialog.getState().request?.run
    try {
      const session = preview?.session ?? ''
      const skip = [...excluded]
      const result = replace
        ? await Replace(game, session, profileId, skip)
        : await Import(game, session, intoOpen ? profileId : '', skip)
      await afterImport(game, replace || intoOpen, result, {
        name: preview?.name,
        settings: preview?.settings ?? 0,
      })
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
  const signIn = () => {
    close()
    importAfterSignIn(signInOptions({ profileId, external, tab, path, text }))
  }

  return {
    external,
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
    previewExternal,
    previewCollectionUpdate,
    paste,
    pick,
    reset,
    dismiss: close,
    toggle,
    run,
    runReplace: () => run(true, true),
    signIn,
  }
}

export type ImportFlow = ReturnType<typeof useImportFlow>
export { shouldOpenQueueAfterImport }
