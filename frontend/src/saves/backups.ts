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
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useSaves } from './store.ts'

export type Status = 'idle' | 'loading' | 'ready' | 'error'

// The list belongs to one game and profile; restore, pin and folder act on that game.
export const useSaveBackups = create<{
  game: string
  profile: string
  items: Backup[]
  status: Status
  error: string
  // Reads the backups folder afresh for game and profile, dropping the rows on screen so none outlive their file.
  load: (game: string, profile: string) => Promise<void>
  // Reads the same game and profile again, keeping the rows on screen until the new list arrives.
  reload: () => Promise<void>
  restore: (name: string, folders: string[] | null) => Promise<void>
  setPinned: (name: string, pinned: boolean) => Promise<void>
  openFolder: () => Promise<void>
}>((set, get) => {
  let latest = 0
  const read = async () => {
    latest += 1
    const id = latest
    set({ status: 'loading', error: '' })
    try {
      const items = (await ListBackups(get().game, get().profile)) ?? []
      if (id === latest) {
        set({ items, status: 'ready' })
      }
    } catch (e) {
      if (id === latest) {
        set({ items: [], status: 'error', error: errorMessage(e) })
      }
    }
  }
  return {
    game: '',
    profile: '',
    items: [],
    status: 'idle',
    error: '',
    load: async (game, profile) => {
      set({ game, profile, items: [] })
      await read()
    },
    reload: async () => {
      if (get().game) {
        await read()
      }
    },
    restore: async (name, folders) => {
      await RestoreBackup(get().game, get().profile, name, folders)
      await Promise.all([get().reload(), useSaves.getState().reload()])
    },
    setPinned: async (name, pinned) => {
      await SetBackupPinned(get().game, name, pinned)
      await get().reload()
    },
    openFolder: async () => {
      try {
        await OpenBackupsFolder(get().game)
      } catch (e) {
        reportUnexpected(e)
      }
    },
  }
})

export function causeLabel(b: Backup, profiles: readonly { id: string; name: string }[]): string {
  if (b.kind === 'update' && b.profile) {
    const name = profiles.find((p) => p.id === b.profile)?.name
    return name === undefined
      ? i18n._(msg`Before updating a deleted profile`)
      : i18n._(msg`Before updating ${name}`)
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
