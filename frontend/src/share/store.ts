import { create } from 'zustand'
import type { ProfilePreview } from '../../bindings/github.com/Rethunk-AI/mortar/internal/migrate/models.ts'
import { Discard } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/service.ts'
import { openSettings, useNav } from '../nav/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'
import type { ModListFormat } from './modList.ts'

let runs = 0

// What Import was showing when it sent the user to sign in to Nexus Mods.
let waiting: ImportOptions | null = null

useNexus.subscribe((state, prev) => {
  if (state.signedIn && !prev.signedIn && waiting) {
    const options = waiting
    waiting = null
    useImportDialog.getState().open(options)
  }
})

// Leaving Settings without signing in drops the wait, so a later sign-in elsewhere does not reopen Import.
useNav.subscribe((state) => {
  if (state.route.name !== 'settings') {
    waiting = null
  }
})

export const useShareDialog = create<{
  profileId: string
  keys: string[]
  listFormat: ModListFormat
  open: (profileId: string, keys?: string[]) => void
  close: () => void
  setListFormat: (listFormat: ModListFormat) => void
}>((set) => ({
  profileId: '',
  keys: [],
  listFormat: 'markdown',
  open: (profileId, keys = []) => set({ profileId, keys }),
  close: () => set({ profileId: '', keys: [] }),
  setListFormat: (listFormat) => set({ listFormat }),
}))

export const openShare = (profileId: string, keys: string[] = []) =>
  useShareDialog.getState().open(profileId, keys)

export interface ImportOptions {
  // The profile "Add to" would fill, when the import was opened from one.
  profileId?: string
  // A link or a .mortar file's path to preview at once.
  link?: string
  file?: string
  data?: string
  external?: ProfilePreview
  collectionUpdate?: boolean
}

export interface ImportRequest {
  profileId: string
  tab: 'link' | 'file' | 'data'
  seed: string
  // Changes on every open, so opening again with the same seed previews again.
  run: number
  external?: ProfilePreview
  collectionUpdate?: boolean
}

export const useImportDialog = create<{
  request: ImportRequest | null
  // An import is running; the dialog stays open until it finishes.
  busy: boolean
  open: (options: ImportOptions) => void
  close: () => void
}>((set) => ({
  request: null,
  busy: false,
  open: ({ profileId = '', link = '', file = '', data = '', external, collectionUpdate }) => {
    runs += 1
    let tab: ImportRequest['tab'] = 'link'
    if (file) {
      tab = 'file'
    }
    if (data) {
      tab = 'data'
    }
    set({
      request: {
        profileId,
        tab,
        seed: data || file || link,
        run: runs,
        ...(external ? { external } : {}),
        ...(collectionUpdate ? { collectionUpdate: true } : {}),
      },
      busy: false,
    })
  },
  close: () => {
    set({ request: null, busy: false })
    Discard().catch(reportUnexpected)
  },
}))

export const openImport = (options: ImportOptions = {}) => useImportDialog.getState().open(options)

// Opens Settings on Nexus Mods; once the account signs in, Import opens again on the same link or file and previews it afresh.
export function importAfterSignIn(options: ImportOptions) {
  waiting = options
  openSettings('nexus')
}
