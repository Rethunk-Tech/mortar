import { create } from 'zustand'
import { Discard } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

let runs = 0

export const useShareDialog = create<{
  profileId: string
  open: (profileId: string) => void
  close: () => void
}>((set) => ({
  profileId: '',
  open: (profileId) => set({ profileId }),
  close: () => set({ profileId: '' }),
}))

export const openShare = (profileId: string) => useShareDialog.getState().open(profileId)

export interface ImportOptions {
  // The profile "Add to" would fill, when the import was opened from one.
  profileId?: string
  // A link or a .mortar file's path to preview at once.
  link?: string
  file?: string
}

export interface ImportRequest {
  profileId: string
  tab: 'link' | 'file'
  seed: string
  // Changes on every open, so opening again with the same seed previews again.
  run: number
}

export const useImportDialog = create<{
  request: ImportRequest | null
  open: (options: ImportOptions) => void
  close: () => void
}>((set) => ({
  request: null,
  open: ({ profileId = '', link = '', file = '' }) => {
    runs += 1
    set({ request: { profileId, tab: file ? 'file' : 'link', seed: file || link, run: runs } })
  },
  close: () => {
    set({ request: null })
    Discard().catch(reportUnexpected)
  },
}))

export const openImport = (options: ImportOptions = {}) => useImportDialog.getState().open(options)
