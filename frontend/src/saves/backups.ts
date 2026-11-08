import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { keepUnchanged } from '@rethunk/react-load-hooks'
import { create } from 'zustand'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import {
  ListBackups,
  OpenBackupsFolder,
  RestoreBackup,
  SetBackupPinned,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { type InlineError, inlineError, reportUnexpected } from '../toasts/report.ts'
import { useSaves } from './store.ts'

export type Status = 'idle' | 'loading' | 'ready' | 'error'

// The list belongs to one game and profile; restore, pin and folder act on that game.
export const useSaveBackups = create<{
  game: string
  profile: string
  items: Backup[]
  // The last list read per game and profile, drawn at once when that pair is opened again.
  lists: Record<string, Backup[]>
  status: Status
  error: InlineError | null
  // Reads the backups folder for game and profile, showing the last list read for that pair until the new one arrives.
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
    const { game, profile } = get()
    const key = `${game}\0${profile}`
    const known = get().lists[key] !== undefined
    set(known ? { error: null } : { status: 'loading', error: null })
    try {
      const fresh = (await ListBackups(game, profile)) ?? []
      const items = keepUnchanged(get().lists[key] ?? [], fresh, (b) => b.name)
      if (id === latest) {
        set({ items, status: 'ready' })
      }
      set((cur) => ({ lists: { ...cur.lists, [key]: items } }))
    } catch (e) {
      if (id === latest) {
        set(
          known
            ? { status: 'error', error: inlineError(e) }
            : { items: [], status: 'error', error: inlineError(e) },
        )
      }
    }
  }
  return {
    game: '',
    profile: '',
    items: [],
    lists: {},
    status: 'idle',
    error: null,
    load: async (game, profile) => {
      const seen = get().lists[`${game}\0${profile}`]
      set({ game, profile, items: seen ?? [], ...(seen ? { status: 'ready' } : {}) })
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
