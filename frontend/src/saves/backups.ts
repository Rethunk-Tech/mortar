import { create } from 'zustand'
import type { Backup } from '../../bindings/github.com/Rethunk-AI/mortar/internal/backup/models.ts'
import {
  ListBackups,
  OpenBackupsFolder,
  RestoreBackup,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'

export type Status = 'idle' | 'loading' | 'ready' | 'error'

export const useSaveBackups = create<{
  items: Backup[]
  status: Status
  error: string
  load: () => Promise<void>
  restore: (name: string, folders: string[] | null) => Promise<void>
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
        const items = (await ListBackups()) ?? []
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
      await RestoreBackup(name, folders)
      await get().load()
    },
    openFolder: async () => {
      try {
        await OpenBackupsFolder()
      } catch (e) {
        reportUnexpected(e)
      }
    },
  }
})

export const getInitialState = () => useSaveBackups.getInitialState()
