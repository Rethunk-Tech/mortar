import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import {
  ListBackups,
  OpenBackupsFolder,
  RestoreBackup,
  SetBackupPinned,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { currentGame } from '../nav/currentGame.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useSaves } from './store.ts'

export type Status = 'idle' | 'loading' | 'ready' | 'error'

export const useSaveBackups = create<{
  items: Backup[]
  status: Status
  error: string
  load: () => Promise<void>
  restore: (name: string, folders: string[] | null) => Promise<void>
  setPinned: (name: string, pinned: boolean) => Promise<void>
  openFolder: () => Promise<void>
}>((set, get) => {
  let latest = 0
  return {
    items: [],
    status: 'idle',
    error: '',
    load: async () => {
      latest += 1
      const id = latest
      set({ status: 'loading', error: '' })
      try {
        const items = (await ListBackups(currentGame())) ?? []
        if (id === latest) {
          set({ items, status: 'ready' })
        }
      } catch (e) {
        if (id === latest) {
          set({ items: [], status: 'error', error: errorMessage(e) })
        }
      }
    },
    restore: async (name, folders) => {
      await RestoreBackup(currentGame(), name, folders)
      await Promise.all([get().load(), useSaves.getState().reload()])
    },
    setPinned: async (name, pinned) => {
      await SetBackupPinned(currentGame(), name, pinned)
      await get().load()
    },
    openFolder: async () => {
      try {
        await OpenBackupsFolder(currentGame())
      } catch (e) {
        reportUnexpected(e)
      }
    },
  }
})

export function causeLabel(b: Backup, profileName: (id: string) => string): string {
  if (b.kind === 'update' && b.profile) {
    return i18n._(msg`Before updating ${profileName(b.profile)}`)
  }
  if (b.kind === 'restore') {
    return i18n._(msg`Before a restore`)
  }
  if (b.kind === 'launch') {
    return i18n._(msg`Before playing`)
  }
  if (b.kind === 'scheduled') {
    return i18n._(msg`Scheduled`)
  }
  if (b.kind === 'manual') {
    return i18n._(msg`Manual`)
  }
  return i18n._(msg`Unknown`)
}
